package glink

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const db_path = "tmp.db"

func TestDbWriteCid(t *testing.T) {
	assert := assert.New(t)
	os.Remove(db_path)

	db, err := NewDb(db_path)
	assert.Nil(err)
	cid1 := db.GetOwnInfo().Uid

	db, err = NewDb(db_path)
	assert.Nil(err)
	cid2 := db.GetOwnInfo().Uid

	os.Remove(db_path)
	assert.Equal(cid1, cid2)
}

func TestDbWriteName(t *testing.T) {
	assert := assert.New(t)
	os.Remove(db_path)

	db, err := NewDb(db_path)
	assert.Nil(err)
	err = db.SetOwnCid("uid")
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

func TestDbWriteMsg(t *testing.T) {
	assert := assert.New(t)
	os.Remove(db_path)
	db, err := NewDb(db_path)
	assert.Nil(err)

	from := "from_cid"
	to := "to_cid"

	msg1 := ChatMessage{FromUid: from, ToCid: to, Index: 1, Text: "test text"}
	msg2 := ChatMessage{FromUid: from + "2", ToCid: to, Index: 1, Text: "test text 2"}
	err = db.SaveMessage(msg1)
	assert.Nil(err)
	err = db.SaveMessage(msg2)
	assert.Nil(err)

	msgs, err := db.GetMessages(from, to, 0, 100000)
	assert.Nil(err)
	assert.Equal(len(msgs), 1)
	assert.Equal(msg1, msgs[0])

	db, err = NewDb(db_path)
	assert.Nil(err)

	msgs, err = db.GetMessages(from+"2", to, 0, 100000)
	assert.Nil(err)
	assert.Equal(len(msgs), 1)
	assert.Equal(msg2, msgs[0])
}

func TestDbWriteMsgIndex(t *testing.T) {
	assert := assert.New(t)
	os.Remove(db_path)
	db, err := NewDb(db_path)
	assert.Nil(err)

	from := "from_cid"
	to := "to_cid"

	msg := ChatMessage{FromUid: from, ToCid: to, Index: 1, Text: "test text"}
	err = db.SaveMessage(msg)
	assert.Nil(err)
	msg = ChatMessage{FromUid: from, ToCid: to, Index: 2, Text: "test text 2"}
	err = db.SaveMessage(msg)
	assert.Nil(err)
	msg = ChatMessage{FromUid: from, ToCid: to, Index: 3, Text: "test text 3"}
	err = db.SaveMessage(msg)
	assert.Nil(err)

	msgs, err := db.GetMessages(from, to, 2, 10)
	assert.Nil(err)
	assert.Equal(len(msgs), 2)
	assert.Equal("test text 2", msgs[0].Text)
	assert.Equal("test text 3", msgs[1].Text)

	os.Remove(db_path)
}

func TestDbHasUid(t *testing.T) {
	assert := assert.New(t)
	os.Remove(db_path)

	db, err := NewDb(db_path)
	assert.Nil(err)

	db.SaveNewUid("uid1", "name", "")

	assert.True(db.IsKnownUid("uid1"))
	assert.False(db.IsKnownUid("uid2"))

	os.Remove(db_path)
}
