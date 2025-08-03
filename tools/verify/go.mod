module verify

go 1.23.0

toolchain go1.24.2

replace gote => ../../

require (
	golang.org/x/term v0.29.0
	gote v0.0.0-00010101000000-000000000000
)

require (
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/sys v0.34.0 // indirect
)
