package amqp

// Copies contexts of ``in`` into new table and returns in.
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
