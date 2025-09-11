package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// WebDAVConfig holds the schema definition for the WebDAVConfig entity.
type WebDAVConfig struct {
	ent.Schema
}

// Fields of the WebDAVConfig.
func (WebDAVConfig) Fields() []ent.Field {
	return []ent.Field{
		field.String("url"),
		field.String("username"),
		field.String("password"),
	}
}

// Edges of the WebDAVConfig.
func (WebDAVConfig) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("storage", Storage.Type).
			Ref("webdav_config").
			Unique(),
	}
}
