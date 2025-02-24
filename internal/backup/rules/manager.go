package rules

// Manager 规则管理器
type Manager struct {
	rules []Rule
}

func NewManager() *Manager {
	m := &Manager{}
	// 注册默认规则
	m.Register(
		NewProfileRule(),
		//NewListRule(),
		//NewContentRule(),
	)
	return m
}

func (m *Manager) Register(rules ...Rule) {
	m.rules = append(m.rules, rules...)
}

func (m *Manager) Handle(ctx *Context) error {
	for _, rule := range m.rules {
		if rule.Match(ctx) {
			if err := rule.Handle(ctx); err != nil {
				return err
			}
		}
	}
	return nil
}
