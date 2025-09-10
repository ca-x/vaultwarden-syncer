package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/edge"
)

type Storage struct {
	ent.Schema
}

func (Storage) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").Unique(),
		field.Enum("type").Values("webdav", "s3"),
		field.JSON("config", map[string]interface{}{}),
		field.Bool("enabled").Default(true),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Storage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("sync_jobs", SyncJob.Type),
	}
}