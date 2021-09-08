package deprecated

import (
	"encoding"
	"encoding/json"
	"github.com/go-redis/redis/v8"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"sync"
	"time"
)

var (
	ErrKeyNotExists = errors.New("Key Not exist.")
	ErrKeyExistsAlready = errors.New("Key exists already.")
	ErrNotConnected = errors.New("Not Connected.")
	ErrNilContext = errors.New("Context is nil value.")
	ErrNoAddress = errors.New("Address Required.")
	ErrNoPort = errors.New("Port Required.")
	ErrUnknown = errors.New("Unknown error occurs")
	ErrUnexpectedToken = errors.New("Unexpected token signing method.")
	ErrInvalidClaims = errors.New("Invalid token claims")
	ErrPasswordHash = errors.New("cannot hash pw")
	ErrNilUserObject = errors.New("Nil user error")
	ErrNoAuthServer = errors.New("No Auth server")
	ErrIncorrectInfo = errors.New("Incorrect mail or password")
)

type RedisData interface {
	Key() string
	Value() RedisValue
}

type RedisValue interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type RedisClient struct {
	*redis.Client
}

func NewRedisClient (opts *redis.Options, timeout ...interface{}) *RedisClient {
	if timeout == nil {
		return &RedisClient{redis.NewClient(opts)}
	} else {
		return &RedisClient{redis.NewClient(opts).WithTimeout(timeout[0].(time.Duration))}
	}
}

func (rc *RedisClient) Ping () (err error) {
	ctx, cancel := context.WithTimeout(context.TODO(), 3000 * time.Millisecond)
	var pong string
	go func (ctx context.Context)  {
		pong, err = rc.Client.Ping(ctx).Result()
		cancel()
		if err != nil {
			err = ErrNotConnected
			return
		}

		if pong != "PONG" {
			err = ErrUnknown
		}

	} (ctx)

	select {
	case <-ctx.Done():
		switch ctx.Err() {
		case context.DeadlineExceeded:
			return ctx.Err()
		case context.Canceled:
			return err
		}
	}

	return nil
}

func (rc *RedisClient) setNX(data RedisData, expiration ...interface{}) error {
	/* if err := rc.Ping(); err != nil {
		return err
	} */
	var bCmd *redis.BoolCmd
	ctx, cancel := context.WithTimeout(context.Background(), rc.Client.Options().WriteTimeout)

	go func (ctx context.Context) {
		defer cancel()
		if expiration == nil {
			bCmd = rc.Client.SetNX(ctx, data.Key(), data.Value(), 0)
		} else {
			bCmd = rc.Client.SetNX(ctx, data.Key(), data.Value(), expiration[0].(time.Duration))
		}
	}(ctx)

	select {
	case <- ctx.Done():
		switch ctx.Err() {
		case context.DeadlineExceeded:
			return ctx.Err()
		case context.Canceled:
			result, err := bCmd.Result()
			if err != nil {
				return err
			}

			if !result {
				return ErrKeyExistsAlready
			} else {
				return nil
			}
		}
	}

	return nil
}

func (rc *RedisClient) get(key string) (*redis.StringCmd, error) {
	/* if err := rc.Ping() ; err != nil {
		return nil, ErrNotConnected
	}*/
	var sCmd *redis.StringCmd
	ctx, cancel := context.WithTimeout(context.Background(), rc.Client.Options().ReadTimeout)

	go func (ctx context.Context) {
		defer cancel()
		sCmd = rc.Client.Get(ctx, key)
	}(ctx)

	select {
	case <- ctx.Done():
		switch ctx.Err() {
		case context.DeadlineExceeded:
			return nil, ctx.Err()
		case context.Canceled:
			if _, err := sCmd.Result(); err != nil {
				return nil, err
			}
		}
	}
	return sCmd, nil
}



type RedisUserData struct {
	user main2.User
}

func NewRedisUserData (user main2.User) *RedisUserData {
	return &RedisUserData{user: user}
}

func (rud *RedisUserData) Key() string {
	return rud.user.ID()
}

func (rud *RedisUserData) Value() RedisValue {
	return rud.user
}

// `RedisUserStore` deprecated
type RedisUserStore struct {
	mtx *sync.RWMutex
	*RedisClient
}

func NewRedisUserStore (opts *redis.Options) (*RedisUserStore, error) {
	// verifying redis opts.
	// stubby.
	// not completed, yet.
	return &RedisUserStore{
		mtx:         &sync.RWMutex{},
		RedisClient: NewRedisClient(opts),
	}, nil
}

func (rus *RedisUserStore) Save (user main2.User) error {
	rus.mtx.Lock()
	defer rus.mtx.Unlock()
	ruser := NewRedisUserData(user)
	if err := rus.setNX(ruser); err != nil {
		return err
	}
	return nil
}

func (rus *RedisUserStore) FindByKey(key string) (main2.User, error) {
	rus.mtx.RLock()
	defer rus.mtx.RUnlock()
	rCmd, err := rus.get(key)
	if err != nil {
		return nil, err
	}
	if marshaled, err := rCmd.Bytes(); err != nil {
		return nil, err
	} else {
		user := &main2.user{}
		if err = json.Unmarshal(marshaled, user); err != nil {
			return nil, err
		} else {
			return user, nil
		}
	}
}