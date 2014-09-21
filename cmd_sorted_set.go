// Commands from http://redis.io/commands#sorted_set

package miniredis

import (
	"strconv"
	"strings"

	"github.com/bsm/redeo"
)

// commandsSortedSet handles all sorted set operations.
func commandsSortedSet(m *Miniredis, srv *redeo.Server) {
	srv.HandleFunc("ZADD", m.cmdZadd)
	srv.HandleFunc("ZCARD", m.cmdZcard)
	// ZCOUNT key min max
	// ZINCRBY key increment member
	// ZINTERSTORE destination numkeys key [key ...] [WEIGHTS weight [weight ...]] [AGGREGATE SUM|MIN|MAX]
	// ZLEXCOUNT key min max
	srv.HandleFunc("ZRANGE", m.cmdZrange)
	// ZRANGEBYLEX key min max [LIMIT offset count]
	// ZRANGEBYSCORE key min max [WITHSCORES] [LIMIT offset count]
	srv.HandleFunc("ZRANK", m.cmdZrank)
	// ZREM key member [member ...]
	// ZREMRANGEBYLEX key min max
	// ZREMRANGEBYRANK key start stop
	// ZREMRANGEBYSCORE key min max
	// ZREVRANGE key start stop [WITHSCORES]
	// ZREVRANGEBYSCORE key max min [WITHSCORES] [LIMIT offset count]
	// ZREVRANK key member
	// ZSCORE key member
	// ZUNIONSTORE destination numkeys key [key ...] [WEIGHTS weight [weight ...]] [AGGREGATE SUM|MIN|MAX]
	// ZSCAN key cursor [MATCH pattern] [COUNT count]
}

// ZADD
func (m *Miniredis) cmdZadd(out *redeo.Responder, r *redeo.Request) error {
	if len(r.Args) < 3 {
		setDirty(r.Client())
		out.WriteErrorString("ERR wrong number of arguments for 'zadd' command")
		return nil
	}

	key := r.Args[0]
	args := r.Args[1:]
	if len(args)%2 != 0 {
		setDirty(r.Client())
		out.WriteErrorString(msgSyntaxError)
		return nil
	}

	elems := map[string]float64{}
	for len(args) > 0 {
		score, err := strconv.ParseFloat(args[0], 64)
		if err != nil {
			setDirty(r.Client())
			out.WriteErrorString("ERR value is not a valid float")
			return nil
		}
		elems[args[1]] = score
		args = args[2:]
	}

	return withTx(m, out, r, func(out *redeo.Responder, ctx *connCtx) {
		db := m.db(ctx.selectedDB)

		if db.exists(key) && db.t(key) != "zset" {
			out.WriteErrorString(ErrWrongType.Error())
			return
		}

		added := 0
		for member, score := range elems {
			if db.zadd(key, score, member) {
				added++
			}
		}
		out.WriteInt(added)
	})
}

// ZCARD
func (m *Miniredis) cmdZcard(out *redeo.Responder, r *redeo.Request) error {
	if len(r.Args) != 1 {
		setDirty(r.Client())
		out.WriteErrorString("ERR wrong number of arguments for 'zcard' command")
		return nil
	}

	key := r.Args[0]

	return withTx(m, out, r, func(out *redeo.Responder, ctx *connCtx) {
		db := m.db(ctx.selectedDB)

		if !db.exists(key) {
			out.WriteZero()
			return
		}

		if db.t(key) != "zset" {
			out.WriteErrorString(ErrWrongType.Error())
			return
		}

		out.WriteInt(db.zcard(key))
	})
}

// ZRANGE
func (m *Miniredis) cmdZrange(out *redeo.Responder, r *redeo.Request) error {
	if len(r.Args) < 3 {
		setDirty(r.Client())
		out.WriteErrorString("ERR wrong number of arguments for 'zrange' command")
		return nil
	}

	key := r.Args[0]
	start, err := strconv.Atoi(r.Args[1])
	if err != nil {
		setDirty(r.Client())
		out.WriteErrorString(msgInvalidInt)
		return nil
	}
	end, err := strconv.Atoi(r.Args[2])
	if err != nil {
		setDirty(r.Client())
		out.WriteErrorString(msgInvalidInt)
		return nil
	}

	withScores := false
	if len(r.Args) > 4 {
		out.WriteErrorString(msgSyntaxError)
		return nil
	}
	if len(r.Args) == 4 {
		if strings.ToLower(r.Args[3]) != "withscores" {
			setDirty(r.Client())
			out.WriteErrorString(msgSyntaxError)
			return nil
		}
		withScores = true
	}

	return withTx(m, out, r, func(out *redeo.Responder, ctx *connCtx) {
		db := m.db(ctx.selectedDB)

		if !db.exists(key) {
			out.WriteBulkLen(0)
			return
		}

		if db.t(key) != "zset" {
			out.WriteErrorString(ErrWrongType.Error())
			return
		}

		members := db.zmembers(key)
		rs, re := redisRange(len(members), start, end, false)
		if withScores {
			out.WriteBulkLen((re - rs) * 2)
		} else {
			out.WriteBulkLen(re - rs)
		}
		for _, el := range members[rs:re] {
			out.WriteString(el)
			if withScores {
				out.WriteString(formatFloat(db.zscore(key, el)))
			}
		}
	})
}

// ZRANK
func (m *Miniredis) cmdZrank(out *redeo.Responder, r *redeo.Request) error {
	if len(r.Args) != 2 {
		setDirty(r.Client())
		out.WriteErrorString("ERR wrong number of arguments for 'zrank' command")
		return nil
	}

	key := r.Args[0]
	member := r.Args[1]

	return withTx(m, out, r, func(out *redeo.Responder, ctx *connCtx) {
		db := m.db(ctx.selectedDB)

		if !db.exists(key) {
			out.WriteNil()
			return
		}

		if db.t(key) != "zset" {
			out.WriteErrorString(ErrWrongType.Error())
			return
		}

		rank, ok := db.zrank(key, member)
		if !ok {
			out.WriteNil()
			return
		}
		out.WriteInt(rank)
	})
}
