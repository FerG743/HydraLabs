package engine

import "fmt"

// Registry maps a block "type" string (as it appears in graph JSON) to the
// factory that builds it. The engine core stays generic; the application
// registers whatever concrete blocks it ships.
type Registry struct {
	factories map[string]Factory
}

func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]Factory)}
}

func (r *Registry) Register(f Factory) {
	r.factories[f.Meta.Type] = f
}

func (r *Registry) Meta(blockType string) (BlockMeta, bool) {
	f, ok := r.factories[blockType]
	return f.Meta, ok
}

func (r *Registry) build(n Node) (Block, error) {
	f, ok := r.factories[n.Type]
	if !ok {
		return nil, fmt.Errorf("unknown block type %q", n.Type)
	}
	return f.New(n.Config)
}
