// Copyright (C) 2026  oito2
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

// Translation of the interface: the message catalogs of the current
// locale, t(key, args) for texts built in JS, and the data-i18n*
// attributes for texts written in the HTML. The locale comes from
// App.GetLanguage() on load and changes live through setLocale (the
// language selector, or the "language-changed" event).
//
// Classic script (not a module): shares one global scope with the other
// scripts under js/.
"use strict";

// Catalog folders merged for a locale, later ones overriding earlier
// ones: the oito2 common base (generated from oito2_ui), the messages
// both applications share, then the application's own.
const I18N_SOURCES = ["./i18n/oito2/", "./i18n/", "./i18n/app/"];

// The locale shown when nothing else is known yet.
const I18N_DEFAULT_LOCALE = "pt-BR";

// Element attributes holding a message key, and what each one sets.
const I18N_ATTRIBUTES = [
  { attr: "data-i18n", apply: function (el, msg) { el.textContent = msg; } },
  { attr: "data-i18n-title", apply: function (el, msg) { el.setAttribute("title", msg); } },
  { attr: "data-i18n-placeholder", apply: function (el, msg) { el.setAttribute("placeholder", msg); } },
  { attr: "data-i18n-aria-label", apply: function (el, msg) { el.setAttribute("aria-label", msg); } },
];

let i18nLocale = I18N_DEFAULT_LOCALE;
let i18nMessages = {};
const i18nCatalogs = {};
const i18nListeners = [];

// Fetches one catalog file; a missing or unreadable one counts as empty.
function fetchCatalog(url) {
  return Promise.resolve().then(function () { return fetch(url); }).then(function (res) {
    return res.ok ? res.json() : {};
  }).catch(function () { return {}; });
}

// Resolves to the merged messages of locale, fetched once per locale.
function loadCatalog(locale) {
  if (!i18nCatalogs[locale]) {
    i18nCatalogs[locale] = Promise.all(I18N_SOURCES.map(function (dir) {
      return fetchCatalog(dir + locale + ".json");
    })).then(function (parts) {
      return Object.assign.apply(null, [{}].concat(parts));
    });
  }
  return i18nCatalogs[locale];
}

// Returns the message key of the current locale with each {name}
// placeholder replaced by args[name]; an unknown key returns the key
// itself, and a placeholder without an argument is kept as is.
function t(key, args) {
  const msg = Object.prototype.hasOwnProperty.call(i18nMessages, key) ? i18nMessages[key] : key;
  if (!args) return msg;
  return msg.replace(/\{([A-Za-z0-9_]+)\}/g, function (m, name) {
    return Object.prototype.hasOwnProperty.call(args, name) ? String(args[name]) : m;
  });
}

// Translates every element under root carrying a data-i18n* attribute.
function applyI18n(root) {
  I18N_ATTRIBUTES.forEach(function (a) {
    root.querySelectorAll("[" + a.attr + "]").forEach(function (el) {
      a.apply(el, t(el.getAttribute(a.attr)));
    });
  });
}

// Registers fn(locale), called after every locale switch, once the page's
// data-i18n* texts are translated — for texts built in JS.
function onLocaleChange(fn) {
  i18nListeners.push(fn);
}

// Switches the interface to locale: loads its catalogs, translates the
// page and notifies the onLocaleChange listeners.
function setLocale(locale) {
  return loadCatalog(locale).then(function (messages) {
    i18nLocale = locale;
    i18nMessages = messages;
    document.documentElement.setAttribute("lang", locale);
    applyI18n(document);
    i18nListeners.forEach(function (fn) { fn(locale); });
  });
}

// Resolves once the saved language is shown; screens that build texts
// with t() at startup wait for it.
const i18nReady = App.GetLanguage().then(function (state) {
  return setLocale((state && state.locale) || I18N_DEFAULT_LOCALE);
}, function () {
  return setLocale(I18N_DEFAULT_LOCALE);
});

// Another window (or the tray) switched the language: follow it.
Events.On("language-changed", function (ev) {
  const locale = ev.data && ev.data.locale;
  if (locale && locale !== i18nLocale) setLocale(locale);
});
