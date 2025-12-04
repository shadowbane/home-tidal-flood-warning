module github.com/shadowbane/home-tidal-flood-warning

go 1.21

require (
	github.com/joho/godotenv v1.5.1
	github.com/julienschmidt/httprouter v1.3.0
	github.com/shadowbane/weather-alert v0.0.0
	go.uber.org/zap v1.27.1
	gorm.io/gorm v1.25.5
)

require (
	github.com/go-sql-driver/mysql v1.7.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/shadowbane/go-logger v0.1.0-alpha // indirect
	go.uber.org/multierr v1.10.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1 // indirect
	gorm.io/driver/mysql v1.5.2 // indirect
)

replace github.com/shadowbane/weather-alert => ../weather-alert
