package gsm

import (
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"go.mongodb.org/mongo-driver/bson"
)

// Session struct stored in MongoDB
type Session struct {
	ID           string     `bson:"_id,omitempty"`
	Data         string     `bson:"data"`
	LastAccessed *time.Time `bson:"lastAccessed"`
}

func (s Session) BSON() bson.M {

	return bson.M{
		"_id":   s.ID,
		"title": s.Data,
		"data":  s.LastAccessed,
	}
}

// MongoStore struct contains options and variables to interact with session settings
type Store struct {
	Codecs     []securecookie.Codec
	Options    *sessions.Options
	Token      Token
	collection string
	// dbSession  *mgo.Session
}

func (*Store) GetToken(r *http.Request, name string) (string, error) {
	panic("implement me")
}

func (*Store) SetToken(w http.ResponseWriter, name, value string, options *sessions.Options) {
	panic("implement me")
}

// func New(s *mgo.Session, collectionName string, options *sessions.Options, ensureTTL bool, keyPairs ...[]byte) *MongoStore {
//
// }
