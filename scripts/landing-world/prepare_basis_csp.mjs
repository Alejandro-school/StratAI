import { readFile, writeFile, copyFile } from 'node:fs/promises';
import { createHash } from 'node:crypto';

// Keep Three r160's exact WASM/API. Replace only the four generated Embind
// JS factories with closures, equivalent to Emscripten DYNAMIC_EXECUTION=0.
// The matching validation compares every texture/mip against the stock glue.
const source = 'frontend/node_modules/three/examples/jsm/libs/basis/';
let code = await readFile(`${source}basis_transcoder.js`, 'utf8');
const sourceHash = createHash('sha256').update(code).digest('hex');
if (sourceHash !== '48a0ef319a28bf0224ee88ded34f74eaf97c175bba9eb18b47fb9720510ad6c4') throw new Error('Upstream Basis glue changed; review the closures before upgrading.');
function replaceFunction(name, replacement) {
  const start = code.indexOf(`function ${name}(`);
  if (start < 0) throw new Error(`Missing upstream function: ${name}`);
  let end = code.indexOf('{', start) + 1, depth = 1, quote = '', escape = false;
  for (; depth; end++) {
    const char = code[end];
    if (quote) {
      if (escape) escape = false;
      else if (char === '\\') escape = true;
      else if (char === quote) quote = '';
    } else if ('\'"`'.includes(char)) quote = char;
    else if (char === '{') depth++;
    else if (char === '}') depth--;
  }
  code = code.slice(0, start) + replacement + code.slice(end);
}
replaceFunction('createNamedFunction', `function createNamedFunction(name,body){
  var wrapper=function(){return body.apply(this,arguments)};
  Object.defineProperty(wrapper,'name',{value:makeLegalFunctionName(name)});
  return wrapper;
}`);
replaceFunction('craftInvokerFunction', `function craftInvokerFunction(humanName,argTypes,classType,cppInvokerFunc,cppTargetFunc){
  var argCount=argTypes.length;
  if(argCount<2)throwBindingError('argTypes array size mismatch');
  var isMethod=argTypes[1]!==null&&classType!==null;
  var needsStack=argTypes.slice(1).some(function(type){return type!==null&&type.destructorFunction===undefined});
  return createNamedFunction(humanName,function(){
    if(arguments.length!==argCount-2)throwBindingError('function '+humanName+' called with '+arguments.length+' arguments, expected '+(argCount-2)+' args!');
    var destructors=needsStack?[]:null;
    var wired=[cppTargetFunc];
    if(isMethod)wired.push(argTypes[1].toWireType(destructors,this));
    for(var i=0;i<arguments.length;i++)wired.push(argTypes[i+2].toWireType(destructors,arguments[i]));
    var value=cppInvokerFunc.apply(null,wired);
    if(needsStack)runDestructors(destructors);
    else for(var i=isMethod?1:2;i<argCount;i++)if(argTypes[i].destructorFunction!==null)argTypes[i].destructorFunction(wired[isMethod?i:i-1]);
    if(argTypes[0].name!=='void')return argTypes[0].fromWireType(value);
  });
}`);
replaceFunction('__emval_get_method_caller', `function __emval_get_method_caller(argCount,argTypes){
  var types=__emval_lookupTypes(argCount,argTypes),retType=types[0];
  return __emval_addMethodCaller(function(handle,name,destructors,args){
    var values=[],offset=0;
    for(var i=1;i<argCount;i++){values.push(types[i].readValueFromPointer(args+offset));offset+=types[i].argPackAdvance}
    var result=handle[name].apply(handle,values);
    for(var i=1;i<argCount;i++)if(types[i].deleteObject)types[i].deleteObject(values[i-1]);
    if(!retType.isVoid)return retType.toWireType(destructors,result);
  });
}`);
replaceFunction('craftEmvalAllocator', `function craftEmvalAllocator(argCount){
  return function(constructor,argTypes,args){
    var values=[];
    for(var i=0;i<argCount;i++){
      var type=requireRegisteredType(Module.HEAP32[(argTypes>>>2)+i],'parameter '+i);
      values.push(type.readValueFromPointer(args));args+=type.argPackAdvance;
    }
    return __emval_register(Reflect.construct(constructor,values));
  };
}`);
code = code.replace('return function(){return Function}()("return this")()', 'return self');
if (/new Function|new_\(Function|newFunc\(Function/.test(code)) throw new Error('Unexpected dynamic execution remains');
await writeFile('frontend/public/basis/basis_transcoder.js', code);
await copyFile(`${source}basis_transcoder.wasm`, 'frontend/public/basis/basis_transcoder.wasm');
console.log(JSON.stringify({ sourceHash, patchedHash: createHash('sha256').update(code).digest('hex') }));
