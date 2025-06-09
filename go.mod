module github.com/godevsig/gshellos

go 1.18

require (
	github.com/fsnotify/fsnotify v1.9.0
	github.com/godevsig/adaptiveservice v0.12.2-0.20250609024628-a3f58512e002
	github.com/godevsig/glib v0.1.2-0.20230830021401-ee447d68739c
	github.com/godevsig/grepo v0.2.5-0.20240409082928-3c01a5227a9a
	github.com/traefik/yaegi v0.15.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/barkimedes/go-deepcopy v0.0.0-20220514131651-17c30cfc62df // indirect
	github.com/go-echarts/go-echarts/v2 v2.2.7 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/gorilla/mux v1.8.0 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/niubaoshu/gotiny v0.0.3 // indirect
	github.com/peterh/liner v1.2.2 // indirect
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/timandy/routine v1.1.3 // indirect
	golang.org/x/sys v0.13.0 // indirect
)

replace (
	github.com/go-echarts/go-echarts/v2 => github.com/godevsig/go-echarts/v2 v2.0.0-20211101104447-e8e4a51bc4fd
	github.com/niubaoshu/gotiny => github.com/godevsig/gotiny v0.0.4-0.20210913173728-083dd4b72177
	github.com/traefik/yaegi => github.com/godevsig/yaegi v0.15.2-0.20240713125216-9ed60ffee13a
)
