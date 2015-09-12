package client

type Collection struct {
	Type         string                 `json:"type,omitempty" yaml:"type,omitempty"`
	ResourceType string                 `json:"resourceType,omitempty" yaml:"resource_type,omitempty"`
	Links        map[string]string      `json:"links,omitempty" yaml:"links,omitempty"`
	CreateTypes  map[string]string      `json:"createTypes,omitempty" yaml:"create_types,omitempty"`
	Actions      map[string]string      `json:"actions,omitempty" yaml:"actions,omitempty"`
	SortLinks    map[string]string      `json:"sortLinks,omitempty" yaml:"sort_links,omitempty"`
	Pagination   *Pagination            `json:"pagination,omitempty" yaml:"pagination,omitempty"`
	Sort         *Sort                  `json:"sort,omitempty" yaml:"sort,omitempty"`
	Filters      map[string][]Condition `json:"filters,omitempty" yaml:"filters,omitempty"`
}

type Sort struct {
	Name    string `json:"name,omitempty" yaml:"name,omitempty"`
	Order   string `json:"order,omitempty" yaml:"order,omitempty"`
	Reverse string `json:"reverse,omitempty" yaml:"reverse,omitempty"`
}

type Condition struct {
	Modifier string      `json:"modifier,omitempty" yaml:"modifier,omitempty"`
	Value    interface{} `json:"value,omitempty" yaml:"value,omitempty"`
}

type Pagination struct {
	Marker   string `json:"marker,omitempty" yaml:"marker,omitempty"`
	First    string `json:"first,omitempty" yaml:"first,omitempty"`
	Previous string `json:"previous,omitempty" yaml:"previous,omitempty"`
	Next     string `json:"next,omitempty" yaml:"next,omitempty"`
	Limit    *int64 `json:"limit,omitempty" yaml:"limit,omitempty"`
	Total    *int64 `json:"total,omitempty" yaml:"total,omitempty"`
	Partial  bool   `json:"partial,omitempty" yaml:"partial,omitempty"`
}

type Resource struct {
	Id      string            `json:"id,omitempty" yaml:"id,omitempty"`
	Type    string            `json:"type,omitempty" yaml:"type,omitempty"`
	Links   map[string]string `json:"links,omitempty" yaml:"links,omitempty"`
	Actions map[string]string `json:"actions,omitempty" yaml:"actions,omitempty"`
}

type Schemas struct {
	Collection
	Data []Schema `json:"data,omitempty" yaml:"data,omitempty"`
}

type Schema struct {
	Resource
	PluralName        string            `json:"pluralName,omitempty" yaml:"plural_name,omitempty"`
	ResourceMethods   []string          `json:"resourceMethods,omitempty" yaml:"resource_methods,omitempty"`
	ResourceFields    map[string]Field  `json:"resourceFields,omitempty" yaml:"resource_fields,omitempty"`
	ResourceActions   map[string]Action `json:"resourceActions,omitempty" yaml:"resource_actions,omitempty"`
	CollectionMethods []string          `json:"collectionMethods,omitempty" yaml:"collection_methods,omitempty"`
	CollectionFields  map[string]Field  `json:"collectionFields,omitempty" yaml:"collection_fields,omitempty"`
	CollectionActions map[string]Action `json:"collectionActions,omitempty" yaml:"collection_actions,omitempty"`
	CollectionFilters map[string]Filter `json:"collectionFilters,omitempty" yaml:"collection_filters,omitempty"`
	IncludeableLinks  []string          `json:"includeableLinks,omitempty" yaml:"includeable_links,omitempty"`
}

type Field struct {
	Type         string      `json:"type,omitempty" yaml:"type,omitempty"`
	Default      interface{} `json:"default,omitempty" yaml:"default,omitempty"`
	Unique       bool        `json:"unique,omitempty" yaml:"unique,omitempty"`
	Nullable     bool        `json:"nullable,omitempty" yaml:"nullable,omitempty"`
	Create       bool        `json:"create,omitempty" yaml:"create,omitempty"`
	Required     bool        `json:"required,omitempty" yaml:"required,omitempty"`
	Update       bool        `json:"update,omitempty" yaml:"update,omitempty"`
	MinLength    *int64      `json:"minLength,omitempty" yaml:"min_length,omitempty"`
	MaxLength    *int64      `json:"maxLength,omitempty" yaml:"max_length,omitempty"`
	Min          *int64      `json:"min,omitempty" yaml:"min,omitempty"`
	Max          *int64      `json:"max,omitempty" yaml:"max,omitempty"`
	Options      []string    `json:"options,omitempty" yaml:"options,omitempty"`
	ValidChars   string      `json:"validChars,omitempty" yaml:"valid_chars,omitempty"`
	InvalidChars string      `json:"invalidChars,omitempty" yaml:"invalid_chars,omitempty"`
}

type Action struct {
	Input  string `json:"input,omitempty" yaml:"input,omitempty"`
	Output string `json:"output,omitempty" yaml:"output,omitempty"`
}

type Filter struct {
	Modifiers []string `json:"modifiers,omitempty" yaml:"modifiers,omitempty"`
}

type ListOpts struct {
	Filters map[string]interface{}
}
