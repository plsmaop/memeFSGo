package helper

import (
	"os/user"
	"strconv"
)

func GetCurUIDAndGID() (uid uint32, gid uint32) {
	if user, err := user.Current(); err == nil {
		if u, err := strconv.ParseUint(user.Uid, 10, 32); err == nil {
			uid = uint32(u)
		}

		if g, err := strconv.ParseUint(user.Gid, 10, 32); err == nil {
			gid = uint32(g)
		}
	}

	return uid, gid
}
