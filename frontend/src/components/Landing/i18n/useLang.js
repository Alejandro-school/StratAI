import React, { createContext, useContext, useState, useCallback } from 'react';
import es from './es.json';
import en from './en.json';

const LANGS = { es, en };
const STORAGE_KEY = 'stratai-lang';

const LangContext = createContext();

const getNestedValue = (obj, path) => {
  return path.split('.').reduce((acc, key) => acc?.[key], obj);
};

export const LangProvider = ({ children }) => {
  const [lang, setLangState] = useState(() => {
    try {
      return localStorage.getItem(STORAGE_KEY) || 'es';
    } catch {
      return 'es';
    }
  });

  const setLang = useCallback((newLang) => {
    if (LANGS[newLang]) {
      setLangState(newLang);
      try { localStorage.setItem(STORAGE_KEY, newLang); } catch {}
    }
  }, []);

  const t = useCallback((key) => {
    return getNestedValue(LANGS[lang], key) || key;
  }, [lang]);

  const toggleLang = useCallback(() => {
    setLang(lang === 'es' ? 'en' : 'es');
  }, [lang, setLang]);

  return (
    <LangContext.Provider value={{ lang, setLang, t, toggleLang }}>
      {children}
    </LangContext.Provider>
  );
};

export const useLang = () => {
  const ctx = useContext(LangContext);
  if (!ctx) throw new Error('useLang must be used within LangProvider');
  return ctx;
};
