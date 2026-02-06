module github.com/sa6mwa/psi/example

go 1.25

require (
	golang.org/x/term v0.39.0
	pkt.systems/emrun v0.5.1
	pkt.systems/psi v0.0.0-00010101000000-000000000000
	pkt.systems/pslog v0.13.3
)

require golang.org/x/sys v0.40.0 // indirect

replace pkt.systems/psi => ../
