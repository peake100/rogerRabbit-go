package amqp

// Store queue declare information for re-establishing queues on disconnect.
type QueueDeclareArgs struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       Table
}

// Arguments passed to channel delete.
type QueueDeleteArgs struct {
	Name     string
	IfUnused bool
	IfEmpty  bool
	NoWait   bool
}

// Store queue bind information for re-establishing bindings on disconnect.
type QueueBindArgs struct {
	Name     string
	Key      string
	Exchange string
	NoWait   bool
	Args     Table
}

type QueueUnbindArgs struct {
	Name     string
	Key      string
	Exchange string
	Args     Table
}

// Store exchange declare information for re-establishing queues on disconnect.
type ExchangeDeclareArgs struct {
	Name       string
	Kind       string
	Durable    bool
	AutoDelete bool
	Internal   bool
	NoWait     bool
	Args       Table
}

type ExchangeDeleteArgs struct {
	Name     string
	IfUnused bool
	NoWait   bool
}

// Store exchange bind information for re-establishing bindings on disconnect.
type ExchangeBindArgs struct {
	Destination string
	Key         string
	Source      string
	NoWait      bool
	Args        Table
}

type ExchangeUnbindArgs struct {
	Destination string
	Key         string
	Source      string
	NoWait      bool
	Args        Table
}
