package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/edge"
)

type SyncJob struct {
	ent.Schema
}

func (SyncJob) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("status").Values("pending", "running", "completed", "failed"),
		field.Enum("operation").Values("backup", "restore"),
		field.Text("message").Optional(),
		field.Time("started_at").Optional(),
		field.Time("completed_at").Optional(),
		field.Time("created_at").Default(time.Now),
	}
}

func (SyncJob) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("storage", Storage.Type).Ref("sync_jobs").Unique(),
	}
}