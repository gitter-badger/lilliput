package register

import (
	"fmt"
	"github.com/pelletier/go-toml"
	"os"
	"reflect"
)

var ini *toml.TomlTree

func init() {

	var err error
	dir, _ := os.Getwd()

	// order in which to search for config file
	files := []string{
		dir + "/dev.ini",
		dir + "/config.ini",
		dir + "/config/dev.ini",
		dir + "/config/config.ini",
	}
	for _, f := range files {
		ini, err = toml.LoadFile(f)
		if err == nil {
			fmt.Println("Loaded Configuration:", f)
			return
		}
	}

	fmt.Println("No configuration file found")
}

func Get(lookup string, def interface{}) interface{} {

	if ini == nil {
		return def
	}

	val := ini.Get(lookup)
	if val == nil {
		return def
	} else {
		return val
	}
}

func GetMap(lookupParent string, def map[string]interface{}) map[string]interface{} {
	if ini == nil {
		return def
	}

	tmp := Get(lookupParent, def)
	t := tmp.(*toml.TomlTree)
	mp := make(map[string]interface{})
	for _, key := range t.Keys() {
		mp[key] = t.Get(key)
	}

	return mp
}

func GetString(lookup string, def string) string {
	value := Get(lookup, def)
	switch value.(type) {
	case string:
		return value.(string)
	}
	return def
}

func GetInt(lookup string, def int) int {
	value := Get(lookup, def)
	switch value.(type) {
	case int, int8, int16, int32, int64:
		return int(reflect.ValueOf(value).Int())
		// don't expect long ints in ini configuraton
		// so converting int64 to int should be ok.
	}
	return def
}

func GetBool(lookup string, def bool) bool {
	value := Get(lookup, def)
	switch value.(type) {
	case bool:
		return value.(bool)
	}
	return def
}
