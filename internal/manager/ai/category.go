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

package ai

import (
	"errors"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/oito2/perci82/internal/config"
	"github.com/oito2/perci82/internal/i18n"
)

// maxCategoryLen is the longest category name, in characters.
const maxCategoryLen = 40

// Categories returns the categories a registered folder can be filed
// under, in alphabetical order.
func Categories() ([]string, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return slices.Clone(cfg.AIContextCategories), nil
}

// categoryIndex returns the index of name in list, compared without
// regard to case, or -1.
func categoryIndex(list []string, name string) int {
	return slices.IndexFunc(list, func(c string) bool { return strings.EqualFold(c, name) })
}

// normalizeCategory trims name and checks its length (1 to maxCategoryLen
// characters) and that it has no line break.
func normalizeCategory(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", i18n.NewError("aic.cat.empty")
	}
	if utf8.RuneCountInString(name) > maxCategoryLen || strings.ContainsAny(name, "\r\n") {
		return "", errors.New(i18n.Tr("aic.cat.tooLong", i18n.Args{"max": maxCategoryLen}))
	}
	return name, nil
}

// sortCategories orders list alphabetically, without regard to case.
func sortCategories(list []string) {
	slices.SortFunc(list, func(a, b string) int {
		if c := strings.Compare(strings.ToLower(a), strings.ToLower(b)); c != 0 {
			return c
		}
		return strings.Compare(a, b)
	})
}

// AddCategory adds the category name and returns it as saved. A name
// already used (without regard to case) is refused.
func AddCategory(name string) (string, error) {
	name, err := normalizeCategory(name)
	if err != nil {
		return "", err
	}
	return name, config.Update(func(cfg *config.Config) error {
		if categoryIndex(cfg.AIContextCategories, name) >= 0 {
			return errors.New(i18n.Tr("aic.cat.exists", i18n.Args{"name": name}))
		}
		cfg.AIContextCategories = append(cfg.AIContextCategories, name)
		sortCategories(cfg.AIContextCategories)
		return nil
	})
}

// RenameCategory renames the category from to name, in the list and in
// every registered folder filed under it. Only a change of case keeps the
// same category; any other name already used is refused.
func RenameCategory(from, name string) error {
	name, err := normalizeCategory(name)
	if err != nil {
		return err
	}
	return config.Update(func(cfg *config.Config) error {
		i := slices.Index(cfg.AIContextCategories, from)
		if i < 0 {
			return errors.New(i18n.Tr("aic.cat.notFound", i18n.Args{"name": from}))
		}
		if j := categoryIndex(cfg.AIContextCategories, name); j >= 0 && j != i {
			return errors.New(i18n.Tr("aic.cat.exists", i18n.Args{"name": name}))
		}
		cfg.AIContextCategories[i] = name
		sortCategories(cfg.AIContextCategories)
		for k := range cfg.AIContexts {
			if cfg.AIContexts[k].Category == from {
				cfg.AIContexts[k].Category = name
			}
		}
		return nil
	})
}

// CategoryUses returns how many registered folders are filed under the
// category name.
func CategoryUses(name string) (int, error) {
	cfg, err := config.Load()
	if err != nil {
		return 0, err
	}
	n := 0
	for _, c := range cfg.AIContexts {
		if c.Category == name {
			n++
		}
	}
	return n, nil
}

// DeleteCategory removes the category name; the registered folders filed
// under it are left without a category. No project file changes.
func DeleteCategory(name string) error {
	return config.Update(func(cfg *config.Config) error {
		i := slices.Index(cfg.AIContextCategories, name)
		if i < 0 {
			return errors.New(i18n.Tr("aic.cat.notFound", i18n.Args{"name": name}))
		}
		cfg.AIContextCategories = slices.Delete(cfg.AIContextCategories, i, i+1)
		for k := range cfg.AIContexts {
			if cfg.AIContexts[k].Category == name {
				cfg.AIContexts[k].Category = ""
			}
		}
		return nil
	})
}

// SetContextCategory files the registered folder at path under category,
// which must be one of the categories, or "" for none.
func SetContextCategory(path, category string) error {
	path, err := cleanContextPath(path)
	if err != nil {
		return err
	}
	return config.Update(func(cfg *config.Config) error {
		if category != "" && !slices.Contains(cfg.AIContextCategories, category) {
			return errors.New(i18n.Tr("aic.cat.notFound", i18n.Args{"name": category}))
		}
		for i, c := range cfg.AIContexts {
			if c.Path == path {
				cfg.AIContexts[i].Category = category
				return nil
			}
		}
		return ErrContextNotFound
	})
}

// CheckCategory returns an error unless category is one of the categories
// or "" (none).
func CheckCategory(category string) error {
	if category == "" {
		return nil
	}
	list, err := Categories()
	if err != nil {
		return err
	}
	if !slices.Contains(list, category) {
		return errors.New(i18n.Tr("aic.cat.notFound", i18n.Args{"name": category}))
	}
	return nil
}

// ensureCategory adds the category name when it is not one yet (compared
// without regard to case) and returns the category's saved name; "" stays
// "". Used by Import for the categories an export file brings.
func ensureCategory(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil
	}
	name, err := normalizeCategory(name)
	if err != nil {
		return "", err
	}
	saved := name
	err = config.Update(func(cfg *config.Config) error {
		if i := categoryIndex(cfg.AIContextCategories, name); i >= 0 {
			saved = cfg.AIContextCategories[i]
			return nil
		}
		cfg.AIContextCategories = append(cfg.AIContextCategories, name)
		sortCategories(cfg.AIContextCategories)
		return nil
	})
	return saved, err
}
