package glink

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const db_path = "tmp.db"

func TestDbCidNotNull(t *testing.T) {
	assert := assert.New(t)

	db, err := NewDb("")
	assert.Nil(err)

	assert.NotEqual(db.GetOwnInfo().Cid, "")
}

func TestDbWriteCid(t *testing.T) {
	assert := assert.New(t)
	os.Remove(db_path)

	db, err := NewDb(db_path)
	assert.Nil(err)
	cid1 := db.GetOwnInfo().Cid

	db, err = NewDb(db_path)
	assert.Nil(err)
	cid2 := db.GetOwnInfo().Cid

	os.Remove(db_path)
	assert.Equal(cid1, cid2)
}

func TestDbWriteName(t *testing.T) {
	assert := assert.New(t)
	os.Remove(db_path)

	db, err := NewDb(db_path)
	assert.Nil(err)

	err = db.SetOwnName("my_name")
	assert.Nil(err)

	name1 := db.GetOwnInfo().Name
	assert.NotEqual(name1, "")

	db, err = NewDb(db_path)
	assert.Nil(err)
	name2 := db.GetOwnInfo().Name

	os.Remove(db_path)
	assert.Equal(name1, name2)
}
