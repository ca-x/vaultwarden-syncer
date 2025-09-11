package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

// S3Config holds the schema definition for the S3Config entity.
type S3Config struct {
	ent.Schema
}

// Fields of the S3Config.
func (S3Config) Fields() []ent.Field {
	return []ent.Field{
		field.String("endpoint").Optional(),
		field.String("access_key_id"),
		field.String("secret_access_key"),
		field.String("region"),
		field.String("bucket"),
	}
}

// Edges of the S3Config.
func (S3Config) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("storage", Storage.Type).
			Ref("s3_config").
			Unique(),
	}
}
