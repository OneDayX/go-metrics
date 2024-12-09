module main

go 1.23.2

replace memstorage => ../../internal/memstorage

replace metrics => ../../internal/metrics

require (
	memstorage v0.0.0-00010101000000-000000000000 // direct
	metrics v0.0.0-00010101000000-000000000000 // direct
)
