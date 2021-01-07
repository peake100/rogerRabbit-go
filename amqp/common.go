package amqp

// copyTable copies contexts of ``in`` into new table and returns it. Table is not
// deep-copied, only shallow-copied.
func copyTable(in Table) Table {
	// If in is nil, return nil.
	if in == nil {
		return nil
	}

	newTable := make(Table, len(in))
	for key, value := range in {
		newTable[key] = value
	}

	return newTable
}
