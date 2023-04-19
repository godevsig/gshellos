module github.com/godevsig/gshellos

go 1.16

require (
	github.com/godevsig/adaptiveservice v0.9.24-0.20230329141516-d104c403854d
	github.com/godevsig/grepo v0.2.2-0.20230329153956-e42313fb6947
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/traefik/yaegi v0.13.0
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
)

replace (
	github.com/go-echarts/go-echarts/v2 => github.com/godevsig/go-echarts/v2 v2.0.0-20211101104447-e8e4a51bc4fd
	github.com/niubaoshu/gotiny => github.com/godevsig/gotiny v0.0.4-0.20210913173728-083dd4b72177
)
