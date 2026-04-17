import i18n from "i18next"
import LanguageDetector from "i18next-browser-languagedetector"
import { initReactI18next } from "react-i18next"

import en from "@/locales/en.json"
import ptBR from "@/locales/pt-BR.json"

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources: {
      en: { translation: en },
      "pt-BR": { translation: ptBR },
      pt: { translation: ptBR },
    },
    fallbackLng: "en",
    supportedLngs: ["en", "pt-BR", "pt"],
    nonExplicitSupportedLngs: true,
    interpolation: {
      escapeValue: false,
    },
    detection: {
      order: ["querystring", "localStorage", "navigator", "htmlTag"],
      lookupQuerystring: "lang",
      lookupLocalStorage: "panel-language",
      caches: ["localStorage"],
    },
  })

export default i18n
