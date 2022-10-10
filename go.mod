module github.com/godevsig/gshellos

go 1.16

require (
	github.com/godevsig/adaptiveservice v0.9.24-0.20221009024351-f0cc63096d9e
	github.com/godevsig/grepo v0.2.2-0.20221010150444-df1f983f673e
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/traefik/yaegi v0.13.0
)

replace (
	github.com/go-echarts/go-echarts/v2 => github.com/godevsig/go-echarts/v2 v2.0.0-20211101104447-e8e4a51bc4fd
	github.com/niubaoshu/gotiny => github.com/godevsig/gotiny v0.0.4-0.20210913173728-083dd4b72177
)
