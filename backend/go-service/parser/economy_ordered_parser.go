package parser

import (
	"reflect"

	dem "github.com/markus-wa/demoinfocs-golang/v5/pkg/demoinfocs"
	dp "github.com/markus-wa/godispatch"
)

// Selector 8 has explicit dependencies between project observers. godispatch
// v1.4.1 builds its first cache from maps, despite its registration-order comment.
// Register one project callback per concrete type and deliver our consumers in
// registration order. This does NOT order SDK internals relative to this callback
// or change packet priority/wire order. Older selectors retain their parser path.
type economyOrderedParser struct {
	dem.Parser
	events, messages map[reflect.Type][]economyOrderedHandler
}

type economyOrderedHandler struct {
	id dp.HandlerIdentifier
	fn reflect.Value
}

func newEconomyOrderedParser(p dem.Parser) *economyOrderedParser {
	return &economyOrderedParser{Parser: p, events: map[reflect.Type][]economyOrderedHandler{}, messages: map[reflect.Type][]economyOrderedHandler{}}
}

func registerEconomyOrdered(groups map[reflect.Type][]economyOrderedHandler, register func(any) dp.HandlerIdentifier, handler any) dp.HandlerIdentifier {
	fn := reflect.ValueOf(handler)
	typ := fn.Type()
	if typ.Kind() != reflect.Func || typ.NumIn() != 1 || typ.NumOut() != 0 || typ.In(0).Kind() == reflect.Interface {
		panic("economy observer requires a concrete single-argument callback")
	}
	input := typ.In(0)
	if _, exists := groups[input]; !exists {
		groups[input] = nil
		register(reflect.MakeFunc(typ, func(args []reflect.Value) []reflect.Value {
			// A dispatch uses its current snapshot; additions apply to the next one.
			for _, h := range groups[input] {
				h.fn.Call(args)
			}
			return nil
		}).Interface())
	}
	id := new(int)
	groups[input] = append(groups[input], economyOrderedHandler{id: id, fn: fn})
	return id
}

func unregisterEconomyOrdered(groups map[reflect.Type][]economyOrderedHandler, id dp.HandlerIdentifier) {
	for typ, handlers := range groups {
		for i, h := range handlers {
			if h.id == id {
				// Preserve an in-flight dispatch's backing array.
				groups[typ] = append(append([]economyOrderedHandler{}, handlers[:i]...), handlers[i+1:]...)
				return
			}
		}
	}
}

func (p *economyOrderedParser) RegisterEventHandler(h any) dp.HandlerIdentifier {
	return registerEconomyOrdered(p.events, p.Parser.RegisterEventHandler, h)
}

func (p *economyOrderedParser) RegisterNetMessageHandler(h any) dp.HandlerIdentifier {
	return registerEconomyOrdered(p.messages, p.Parser.RegisterNetMessageHandler, h)
}

func (p *economyOrderedParser) UnregisterEventHandler(id dp.HandlerIdentifier) {
	unregisterEconomyOrdered(p.events, id)
}

func (p *economyOrderedParser) UnregisterNetMessageHandler(id dp.HandlerIdentifier) {
	unregisterEconomyOrdered(p.messages, id)
}
