package valkey

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"github.com/grafana/sobek"
	valkey "github.com/valkey-io/valkey-go"
	"go.k6.io/k6/js/common"
	"go.k6.io/k6/js/modules"
	"go.k6.io/k6/js/promises"
)

// Client represents the Client constructor (i.e. `new redis.Client()`) and
// returns a new Redis client object.
type Client struct {
	vu            modules.VU
	valkeyOptions valkey.ClientOption
	valkeyClient  valkey.Client
}

// stringifyValue converts any value to its string representation.
func stringifyValue(v any) string {
	return fmt.Sprint(v)
}

// stringifyAll converts a slice of any values to a slice of strings.
func stringifyAll(vals []any) []string {
	result := make([]string, len(vals))
	for i, v := range vals {
		result[i] = stringifyValue(v)
	}
	return result
}

// resolveMessage converts a ValkeyMessage into a Go value suitable for
// resolving a JS promise.
func resolveMessage(msg valkey.ValkeyMessage) any {
	if msg.IsNil() {
		return nil
	}
	if n, err := msg.AsInt64(); err == nil {
		return n
	}
	if s, err := msg.ToString(); err == nil {
		return s
	}
	if arr, err := msg.ToArray(); err == nil {
		result := make([]any, len(arr))
		for i, m := range arr {
			result[i] = resolveMessage(m)
		}
		return result
	}
	return nil
}

// Set the given key with the given value.
//
// If the provided value is not a supported type, the promise is rejected with an error.
//
// The value for `expiration` is interpreted as seconds.
func (c *Client) Set(key string, value any, expiration int) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	if err := c.isSupportedType(1, value); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		var cmd valkey.Completed
		if expiration > 0 {
			cmd = c.valkeyClient.B().Set().Key(key).Value(stringifyValue(value)).ExSeconds(int64(expiration)).Build()
		} else {
			cmd = c.valkeyClient.B().Set().Key(key).Value(stringifyValue(value)).Build()
		}
		err := c.valkeyClient.Do(ctx, cmd).Error()
		if err != nil {
			reject(err)
			return
		}

		resolve("OK")
	}()

	return promise
}

// Get returns the value for the given key.
//
// If the key does not exist, the promise is rejected with an error.
//
// If the key does not exist, the promise is rejected with an error.
func (c *Client) Get(key string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		value, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Get().Key(key).Build()).ToString()
		if err != nil {
			reject(err)
			return
		}

		resolve(value)
	}()

	return promise
}

// GetSet sets the value of key to value and returns the old value stored
//
// If the provided value is not a supported type, the promise is rejected with an error.
func (c *Client) GetSet(key string, value any) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	if err := c.isSupportedType(1, value); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		oldValue, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Getset().Key(key).Value(stringifyValue(value)).Build()).ToString()
		if err != nil {
			reject(err)
			return
		}

		resolve(oldValue)
	}()

	return promise
}

// Del removes the specified keys. A key is ignored if it does not exist
func (c *Client) Del(keys ...string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		n, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Del().Key(keys...).Build()).AsInt64()
		if err != nil {
			reject(err)
			return
		}

		resolve(n)
	}()

	return promise
}

// GetDel gets the value of key and deletes the key.
//
// If the key does not exist, the promise is rejected with an error.
func (c *Client) GetDel(key string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		value, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Getdel().Key(key).Build()).ToString()
		if err != nil {
			reject(err)
			return
		}

		resolve(value)
	}()

	return promise
}

// Exists returns the number of key arguments that exist.
// Note that if the same existing key is mentioned in the argument
// multiple times, it will be counted multiple times.
func (c *Client) Exists(keys ...string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		n, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Exists().Key(keys...).Build()).AsInt64()
		if err != nil {
			reject(err)
			return
		}

		resolve(n)
	}()

	return promise
}

// Incr increments the number stored at `key` by one. If the key does
// not exist, it is set to zero before performing the operation. An
// error is returned if the key contains a value of the wrong type, or
// contains a string that cannot be represented as an integer.
func (c *Client) Incr(key string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		newValue, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Incr().Key(key).Build()).AsInt64()
		if err != nil {
			reject(err)
			return
		}

		resolve(newValue)
	}()

	return promise
}

// IncrBy increments the number stored at `key` by `increment`. If the key does
// not exist, it is set to zero before performing the operation. An
// error is returned if the key contains a value of the wrong type, or
// contains a string that cannot be represented as an integer.
func (c *Client) IncrBy(key string, increment int64) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		newValue, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Incrby().Key(key).Increment(increment).Build()).AsInt64()
		if err != nil {
			reject(err)
			return
		}

		resolve(newValue)
	}()

	return promise
}

// Decr decrements the number stored at `key` by one. If the key does
// not exist, it is set to zero before performing the operation. An
// error is returned if the key contains a value of the wrong type, or
// contains a string that cannot be represented as an integer.
func (c *Client) Decr(key string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		newValue, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Decr().Key(key).Build()).AsInt64()
		if err != nil {
			reject(err)
			return
		}

		resolve(newValue)
	}()

	return promise
}

// DecrBy decrements the number stored at `key` by `decrement`. If the key does
// not exist, it is set to zero before performing the operation. An
// error is returned if the key contains a value of the wrong type, or
// contains a string that cannot be represented as an integer.
func (c *Client) DecrBy(key string, decrement int64) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		newValue, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Decrby().Key(key).Decrement(decrement).Build()).AsInt64()
		if err != nil {
			reject(err)
			return
		}

		resolve(newValue)
	}()

	return promise
}

// RandomKey returns a random key.
//
// If the database is empty, the promise is rejected with an error.
func (c *Client) RandomKey() *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		key, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Randomkey().Build()).ToString()
		if err != nil {
			reject(err)
			return
		}

		resolve(key)
	}()

	return promise
}

// Mget returns the values associated with the specified keys.
func (c *Client) Mget(keys ...string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		messages, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Mget().Key(keys...).Build()).ToArray()
		if err != nil {
			reject(err)
			return
		}

		values := make([]any, len(messages))
		for i, msg := range messages {
			if msg.IsNil() {
				values[i] = nil
			} else {
				s, sErr := msg.ToString()
				if sErr != nil {
					values[i] = nil
				} else {
					values[i] = s
				}
			}
		}

		resolve(values)
	}()

	return promise
}

// Expire sets a timeout on key, after which the key will automatically
// be deleted.
// Note that calling Expire with a non-positive timeout will result in
// the key being deleted rather than expired.
func (c *Client) Expire(key string, seconds int) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		ok, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Expire().Key(key).Seconds(int64(seconds)).Build()).AsBool()
		if err != nil {
			reject(err)
			return
		}

		resolve(ok)
	}()

	return promise
}

// Ttl returns the remaining time to live of a key that has a timeout.
//
//nolint:revive
func (c *Client) Ttl(key string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		ttlSeconds, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Ttl().Key(key).Build()).AsInt64()
		if err != nil {
			reject(err)
			return
		}

		resolve(float64(ttlSeconds))
	}()

	return promise
}

// Persist removes the existing timeout on key.
func (c *Client) Persist(key string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		ok, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Persist().Key(key).Build()).AsBool()
		if err != nil {
			reject(err)
			return
		}

		resolve(ok)
	}()

	return promise
}

// Lpush inserts all the specified values at the head of the list stored
// at `key`. If `key` does not exist, it is created as empty list before
// performing the push operations. When `key` holds a value that is not
// a list, and error is returned.
func (c *Client) Lpush(key string, values ...any) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	if err := c.isSupportedType(1, values...); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		listLength, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Lpush().Key(key).Element(stringifyAll(values)...).Build()).AsInt64()
		if err != nil {
			reject(err)
			return
		}

		resolve(listLength)
	}()

	return promise
}

// Rpush inserts all the specified values at the tail of the list stored
// at `key`. If `key` does not exist, it is created as empty list before
// performing the push operations.
func (c *Client) Rpush(key string, values ...any) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	if err := c.isSupportedType(1, values...); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		listLength, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Rpush().Key(key).Element(stringifyAll(values)...).Build()).AsInt64()
		if err != nil {
			reject(err)
			return
		}

		resolve(listLength)
	}()

	return promise
}

// Lpop removes and returns the first element of the list stored at `key`.
//
// If the list does not exist, this command rejects the promise with an error.
func (c *Client) Lpop(key string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		value, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Lpop().Key(key).Build()).ToString()
		if err != nil {
			reject(err)
			return
		}

		resolve(value)
	}()

	return promise
}

// Rpop removes and returns the last element of the list stored at `key`.
//
// If the list does not exist, this command rejects the promise with an error.
func (c *Client) Rpop(key string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		value, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Rpop().Key(key).Build()).ToString()
		if err != nil {
			reject(err)
			return
		}

		resolve(value)
	}()

	return promise
}

// Lrange returns the specified elements of the list stored at `key`. The
// offsets start and stop are zero-based indexes. These offsets can be
// negative numbers, where they indicate offsets starting at the end of
// the list.
func (c *Client) Lrange(key string, start, stop int64) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		values, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Lrange().Key(key).Start(start).Stop(stop).Build()).AsStrSlice()
		if err != nil {
			reject(err)
			return
		}

		resolve(values)
	}()

	return promise
}

// Lindex returns the specified element of the list stored at `key`.
// The index is zero-based. Negative indices can be used to designate
// elements starting at the tail of the list.
//
// If the list does not exist, this command rejects the promise with an error.
func (c *Client) Lindex(key string, index int64) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		value, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Lindex().Key(key).Index(index).Build()).ToString()
		if err != nil {
			reject(err)
			return
		}

		resolve(value)
	}()

	return promise
}

// Lset sets the list element at `index` to `element`.
//
// If the list does not exist, this command rejects the promise with an error.
func (c *Client) Lset(key string, index int64, element string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Lset().Key(key).Index(index).Element(element).Build()).Error()
		if err != nil {
			reject(err)
			return
		}

		resolve("OK")
	}()

	return promise
}

// Lrem removes the first `count` occurrences of `value` from the list stored
// at `key`. If `count` is positive, elements are removed from the beginning of the list.
// If `count` is negative, elements are removed from the end of the list.
// If `count` is zero, all elements matching `value` are removed.
//
// If the list does not exist, this command rejects the promise with an error.
func (c *Client) Lrem(key string, count int64, value string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		n, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Lrem().Key(key).Count(count).Element(value).Build()).AsInt64()
		if err != nil {
			reject(err)
			return
		}

		resolve(n)
	}()

	return promise
}

// Llen returns the length of the list stored at `key`. If `key`
// does not exist, it is interpreted as an empty list and 0 is returned.
//
// If the list does not exist, this command rejects the promise with an error.
func (c *Client) Llen(key string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		length, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Llen().Key(key).Build()).AsInt64()
		if err != nil {
			reject(err)
			return
		}

		resolve(length)
	}()

	return promise
}

// Hset sets the specified field in the hash stored at `key` to `value`.
// If the `key` does not exist, a new key holding a hash is created.
// If `field` already exists in the hash, it is overwritten.
//
// If the hash does not exist, this command rejects the promise with an error.
func (c *Client) Hset(key string, field string, value any) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	if err := c.isSupportedType(2, value); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		n, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Hset().Key(key).FieldValue().FieldValue(field, stringifyValue(value)).Build()).AsInt64()
		if err != nil {
			reject(err)
			return
		}

		resolve(n)
	}()

	return promise
}

// Hsetnx sets the specified field in the hash stored at `key` to `value`,
// only if `field` does not yet exist. If `key` does not exist, a new key
// holding a hash is created. If `field` already exists, this operation
// has no effect.
func (c *Client) Hsetnx(key, field, value string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		ok, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Hsetnx().Key(key).Field(field).Value(value).Build()).AsBool()
		if err != nil {
			reject(err)
			return
		}

		resolve(ok)
	}()

	return promise
}

// Hget returns the value associated with `field` in the hash stored at `key`.
//
// If the hash does not exist, this command rejects the promise with an error.
func (c *Client) Hget(key, field string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		value, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Hget().Key(key).Field(field).Build()).ToString()
		if err != nil {
			reject(err)
			return
		}

		resolve(value)
	}()

	return promise
}

// Hdel deletes the specified fields from the hash stored at `key`.
func (c *Client) Hdel(key string, fields ...string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		n, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Hdel().Key(key).Field(fields...).Build()).AsInt64()
		if err != nil {
			reject(err)
			return
		}

		resolve(n)
	}()

	return promise
}

// Hgetall returns all fields and values of the hash stored at `key`.
//
// If the hash does not exist, this command rejects the promise with an error.
func (c *Client) Hgetall(key string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		hashMap, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Hgetall().Key(key).Build()).AsStrMap()
		if err != nil {
			reject(err)
			return
		}

		resolve(hashMap)
	}()

	return promise
}

// Hkeys returns all fields of the hash stored at `key`.
//
// If the hash does not exist, this command rejects the promise with an error.
func (c *Client) Hkeys(key string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		keys, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Hkeys().Key(key).Build()).AsStrSlice()
		if err != nil {
			reject(err)
			return
		}

		resolve(keys)
	}()

	return promise
}

// Hvals returns all values of the hash stored at `key`.
//
// If the hash does not exist, this command rejects the promise with an error.
func (c *Client) Hvals(key string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		values, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Hvals().Key(key).Build()).AsStrSlice()
		if err != nil {
			reject(err)
			return
		}

		resolve(values)
	}()

	return promise
}

// Hlen returns the number of fields in the hash stored at `key`.
//
// If the hash does not exist, this command rejects the promise with an error.
func (c *Client) Hlen(key string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		n, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Hlen().Key(key).Build()).AsInt64()
		if err != nil {
			reject(err)
			return
		}

		resolve(n)
	}()

	return promise
}

// Hincrby increments the integer value of `field` in the hash stored at `key`
// by `increment`. If `key` does not exist, a new key holding a hash is created.
// If `field` does not exist the value is set to 0 before the operation is
// set to 0 before the operation is performed.
func (c *Client) Hincrby(key, field string, increment int64) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		newValue, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Hincrby().Key(key).Field(field).Increment(increment).Build()).AsInt64()
		if err != nil {
			reject(err)
			return
		}

		resolve(newValue)
	}()

	return promise
}

// Sadd adds the specified members to the set stored at key.
// Specified members that are already a member of this set are ignored.
// If key does not exist, a new set is created before adding the specified members.
func (c *Client) Sadd(key string, members ...any) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	if err := c.isSupportedType(1, members...); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		n, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Sadd().Key(key).Member(stringifyAll(members)...).Build()).AsInt64()
		if err != nil {
			reject(err)
			return
		}

		resolve(n)
	}()

	return promise
}

// Srem removes the specified members from the set stored at key.
// Specified members that are not a member of this set are ignored.
// If key does not exist, it is treated as an empty set and this command returns 0.
func (c *Client) Srem(key string, members ...any) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	if err := c.isSupportedType(1, members...); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		n, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Srem().Key(key).Member(stringifyAll(members)...).Build()).AsInt64()
		if err != nil {
			reject(err)
			return
		}

		resolve(n)
	}()

	return promise
}

// Sismember returns if member is a member of the set stored at key.
func (c *Client) Sismember(key string, member any) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	if err := c.isSupportedType(1, member); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		ok, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Sismember().Key(key).Member(stringifyValue(member)).Build()).AsBool()
		if err != nil {
			reject(err)
			return
		}

		resolve(ok)
	}()

	return promise
}

// Smembers returns all members of the set stored at key.
func (c *Client) Smembers(key string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		members, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Smembers().Key(key).Build()).AsStrSlice()
		if err != nil {
			reject(err)
			return
		}

		resolve(members)
	}()

	return promise
}

// Srandmember returns a random element from the set value stored at key.
//
// If the set does not exist, the promise is rejected with an error.
func (c *Client) Srandmember(key string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		element, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Srandmember().Key(key).Build()).ToString()
		if err != nil {
			reject(err)
			return
		}

		resolve(element)
	}()

	return promise
}

// Spop removes and returns a random element from the set value stored at key.
//
// If the set does not exist, the promise is rejected with an error.
func (c *Client) Spop(key string) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		element, err := c.valkeyClient.Do(ctx, c.valkeyClient.B().Spop().Key(key).Build()).ToString()
		if err != nil {
			reject(err)
			return
		}

		resolve(element)
	}()

	return promise
}

// SendCommand sends a command to the redis server.
func (c *Client) SendCommand(command string, args ...any) *sobek.Promise {
	promise, resolve, reject := promises.New(c.vu)

	if err := c.connect(); err != nil {
		reject(err)
		return promise
	}

	if err := c.isSupportedType(1, args...); err != nil {
		reject(err)
		return promise
	}

	go func() {
		ctx := c.vu.Context()
		stringArgs := stringifyAll(args)
		cmd := c.valkeyClient.B().Arbitrary(command).Args(stringArgs...).Build()
		msg, err := c.valkeyClient.Do(ctx, cmd).ToMessage()
		if err != nil {
			reject(err)
			return
		}

		resolve(resolveMessage(msg))
	}()

	return promise
}

// connect establishes the client's connection to the target
// valkey instance(s).
func (c *Client) connect() error {
	// A nil VU state indicates we are in the init context.
	// As a general convention, k6 should not perform IO in the
	// init context. Thus, the Connect method will error if
	// called in the init context.
	vuState := c.vu.State()
	if vuState == nil {
		return common.NewInitContextError("connecting to a valkey server in the init context is not supported")
	}

	// If the valkeyClient is already instantiated, it is safe
	// to assume that the connection is already established.
	if c.valkeyClient != nil {
		return nil
	}

	tlsCfg := c.valkeyOptions.TLSConfig
	if tlsCfg != nil && vuState.TLSConfig != nil {
		// Merge k6 TLS configuration with the one we received from the
		// Client constructor. This will need adjusting depending on which
		// options we want to expose in the module, and how we want
		// the override to work.
		tlsCfg.InsecureSkipVerify = vuState.TLSConfig.InsecureSkipVerify
		tlsCfg.CipherSuites = vuState.TLSConfig.CipherSuites
		tlsCfg.MinVersion = vuState.TLSConfig.MinVersion
		tlsCfg.MaxVersion = vuState.TLSConfig.MaxVersion
		tlsCfg.Renegotiation = vuState.TLSConfig.Renegotiation
		tlsCfg.KeyLogWriter = vuState.TLSConfig.KeyLogWriter
		tlsCfg.Certificates = append(tlsCfg.Certificates, vuState.TLSConfig.Certificates...)

		// Merge Root CAs: start from the VU pool (if any) and add the
		// client-provided CAs on top so both are trusted.
		if vuState.TLSConfig.RootCAs != nil {
			merged := vuState.TLSConfig.RootCAs.Clone()
			if tlsCfg.RootCAs != nil {
				for _, cert := range tlsCfg.RootCAs.Subjects() { //nolint:staticcheck
					merged.AppendCertsFromPEM(cert)
				}
			}
			tlsCfg.RootCAs = merged
		}

		// In order to preserve the underlying effects of the [netext.Dialer], such
		// as handling blocked hostnames, or handling hostname resolution, we override
		// the client's dialer with our own function which uses the VU's [netext.Dialer]
		// and manually upgrades the connection to TLS.
		//
		// See Pull Request's #17 [discussion] for more details.
		//
		// [discussion]: https://github.com/grafana/xk6-redis/pull/17#discussion_r1369707388
		c.valkeyOptions.DialCtxFn = func(ctx context.Context, addr string, dialer *net.Dialer, config *tls.Config) (net.Conn, error) {
			rawConn, err := vuState.Dialer.DialContext(ctx, "tcp", addr)
			if err != nil {
				return nil, err
			}
			tlsConn := tls.Client(rawConn, config)
			if err := tlsConn.HandshakeContext(ctx); err != nil {
				rawConn.Close()
				return nil, err
			}
			return tlsConn, nil
		}
	} else {
		c.valkeyOptions.DialCtxFn = func(ctx context.Context, addr string, dialer *net.Dialer, config *tls.Config) (net.Conn, error) {
			return vuState.Dialer.DialContext(ctx, "tcp", addr)
		}
	}

	client, err := valkey.NewClient(c.valkeyOptions)
	if err != nil {
		return err
	}
	c.valkeyClient = client

	return nil
}

// IsConnected returns true if the client is connected to valkey.
func (c *Client) IsConnected() bool {
	return c.valkeyClient != nil
}

// isSupportedType returns whether the provided arguments are of a type
// supported by the client.
//
// Errors will indicate the zero-indexed position of the argument of
// an unsuppoprted type.
//
// isSupportedType should report type errors with arguments in the correct
// position. To be able to accurately report the argument position in the larger
// context of a call to a redis function, the `offset` argument allows to indicate
// the amount of arguments present in front of the ones we provide to `isSupportedType`.
// For instance, when calling `set`, which takes a key, and a value argument,
// isSupportedType applied to the value should eventually report an error with
// the argument in position 1.
func (c *Client) isSupportedType(offset int, args ...any) error {
	for idx, arg := range args {
		switch arg.(type) {
		case string, int, int64, float64, bool:
			continue
		default:
			return fmt.Errorf(
				"unsupported type provided for argument at index %d, "+
					"supported types are string, number, and boolean", idx+offset)
		}
	}

	return nil
}
