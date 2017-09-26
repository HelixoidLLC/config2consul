package injest

import (
	"config2consul/log"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestInjestKV(t *testing.T) {
	log.SetLevel(log.ErrorLevel)

	consul, deferFn, err := createTestProject("../testing/integration/consul_base/docker-compose.yml", "ssl/ca.crt", "ssl/consul_client.crt", "ssl/consul_client.key")
	if err != nil {
		t.Fatal(err)
	}
	defer deferFn()

	Convey("Controlling KV entries", t, func() {
		Convey("A KV entry is created", func() {
			keyPath := "aa/blah"
			value := "boom"
			configData := consulConfig{
				KeyValue: map[string]interface{}{
					keyPath: value,
				},
			}
			importConfig(consul, &configData)

			result := GetValue(t, consul, keyPath)

			So(string(result.Value), ShouldEqual, value)
		})

		Convey("A boolean KV entry is created as string", func() {
			keyPath := "aa/blah"
			value := true
			configData := consulConfig{
				KeyValue: map[string]interface{}{
					keyPath: value,
				},
			}
			importConfig(consul, &configData)

			result := GetValue(t, consul, keyPath)

			So(string(result.Value), ShouldEqual, "true")
		})

		Convey("An integer KV entry is created as string", func() {
			keyPath1 := "aa/key1"
			keyPath2 := "aa/key2"
			keyPath3 := "aa/key3"
			configData := consulConfig{
				KeyValue: map[string]interface{}{
					keyPath1: 123,
					keyPath2: -123,
					keyPath3: 1.23,
				},
			}
			importConfig(consul, &configData)

			So(string(GetValue(t, consul, keyPath1).Value), ShouldEqual, "123")
			So(string(GetValue(t, consul, keyPath2).Value), ShouldEqual, "-123")
			So(string(GetValue(t, consul, keyPath3).Value), ShouldEqual, "1.23")
		})

		Convey("A KV is overwritten if existed", func() {
			keyPath := "bb/blah"
			value := "foo"

			CreateKV(t, consul, keyPath, []byte("bar"))

			configData := consulConfig{
				KeyValue: map[string]interface{}{
					keyPath: value,
				},
			}
			importConfig(consul, &configData)

			result := GetValue(t, consul, keyPath)

			So(string(result.Value), ShouldEqual, value)
		})

		Convey("A KV is deleted if not defined by a policy", func() {
			keyPath := "cc/blah"

			CreateKV(t, consul, keyPath, []byte("foo"))

			configData := consulConfig{
				KeyValue: map[string]interface{}{
					"dd/blah": "test",
				},
			}

			importConfig(consul, &configData)

			result := GetValue(t, consul, keyPath)

			So(result, ShouldBeNil)
		})

		Convey("A KV tree is ignored if said so", func() {
			keyPath := "ee/blah"
			value := "foo"

			CreateKV(t, consul, keyPath, []byte(value))

			configData := consulConfig{
				KeyValue: map[string]interface{}{
					"ee/": "${ignore}",
				},
			}
			importConfig(consul, &configData)

			result := GetValue(t, consul, keyPath)

			So(string(result.Value), ShouldEqual, value)
		})

		Convey("A tree key have to have a / at the end to differentiate path from a value", func() {
			configData := consulConfig{
				KeyValue: map[string]interface{}{
					"dev": map[interface{}]interface{}{
						"a": "b",
					},
				},
			}
			err := importConfig(consul, &configData)
			So(err, ShouldNotBeNil)
		})

		Convey("A key is treated as a tree when the value is a map", func() {
			configData := consulConfig{
				KeyValue: map[string]interface{}{
					"a/": map[interface{}]interface{}{
						"b": map[interface{}]interface{}{
							"c": "d",
						},
						"e": "f",
					},
				},
			}
			err := importConfig(consul, &configData)
			So(err, ShouldBeNil)

			result := GetValue(t, consul, "a/b/c")
			So(string(result.Value), ShouldEqual, "d")
		})
	})
}
