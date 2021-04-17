package rproducer

// Opts holds options for Producer.
type Opts struct {
	// Whether to confirm publications with the broker.
	confirmPublish bool
	// The buffer size of our internal publication queues.
	internalQueueCapacity int
}

// WithConfirmPublish sets whether to confirm publications with the broker before
// returning on a "Publish" call When true, all called to Producer.Publish() will block
// until a publish confirmation is received from the broker.
//
// Default: true.
func (opts *Opts) WithConfirmPublish(confirm bool) *Opts {
	opts.confirmPublish = confirm
	return opts
}

// WithInternalQueueCapacity sets the internal queue size.
//
// Internally, the producer stores incoming publication requests and published requests
// waiting for a broker acknowledgement in go channels. This options sets the size
// for those internal queues.
//
// Default: 64
func (opts *Opts) WithInternalQueueCapacity(size int) *Opts {
	opts.internalQueueCapacity = size
	return opts
}

// NewOpts returns a new Opts with default options.
func NewOpts() *Opts {
	return new(Opts).
		WithConfirmPublish(true).
		WithInternalQueueCapacity(64)
}
