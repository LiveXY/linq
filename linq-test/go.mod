module linq_benchmark

go 1.25

require (
	github.com/ahmetb/go-linq/v3 v3.2.0
	github.com/livexy/linq v1.1.8
	github.com/samber/lo v1.53.0
)

require golang.org/x/text v0.34.0 // indirect

replace github.com/livexy/linq => ../
