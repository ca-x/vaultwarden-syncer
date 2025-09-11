package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// Storage holds the schema definition for the Storage entity.
type Storage struct {
	ent.Schema
}

// Fields of the Storage.
func (Storage) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").Unique(),
		field.Enum("type").Values("webdav", "s3"),
		field.Bool("enabled").Default(true),
		field.Time("created_at").Default(time.Now),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the Storage.
func (Storage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("sync_jobs", SyncJob.Type),
		edge.To("webdav_config", WebDAVConfig.Type).Unique(),
		edge.To("s3_config", S3Config.Type).Unique(),
	}
}
