module github.com/godevsig/gshellos

go 1.18

require (
	github.com/godevsig/adaptiveservice v0.9.24-0.20230419040517-0c061e57ed21
	github.com/godevsig/grepo v0.2.2-0.20230329153956-e42313fb6947
	github.com/traefik/yaegi v0.14.3
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c
)

require (
	github.com/barkimedes/go-deepcopy v0.0.0-20200817023428-a044a1957ca4 // indirect
	github.com/go-echarts/go-echarts/v2 v2.0.0-20210921152819-048776e902c7 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/niubaoshu/gotiny v0.0.3 // indirect
	github.com/peterh/liner v1.2.1 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
)

replace (
	github.com/go-echarts/go-echarts/v2 => github.com/godevsig/go-echarts/v2 v2.0.0-20211101104447-e8e4a51bc4fd
	github.com/niubaoshu/gotiny => github.com/godevsig/gotiny v0.0.4-0.20210913173728-083dd4b72177
)
