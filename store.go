package gsm

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	lastAccessedField = "lastAccessed"
)

var (
	ErrInvalidLastAccessedTime = errors.New("invalid last accessed time")
)

func New(ctx context.Context, collection *mongo.Collection, options *sessions.Options, keyPairs ...[]byte) *Store {

	return &Store{
		codecs:     securecookie.CodecsFromPairs(keyPairs...),
		options:    options,
		token:      &CookieToken{},
		collection: collection,
		ctx:        ctx,
	}
}

// Session struct stored in MongoDB
type SessionRow struct {
	ID           string    `bson:"_id"`
	UserID       int       `bson:"user_id"`
	Data         string    `bson:"data"`
	LastAccessed time.Time `bson:"last_accessed"`
}

func (s SessionRow) bson() bson.M {

	return bson.M{
		"_id":           s.ID,
		"user_id":       s.UserID,
		"data":          s.Data,
		"last_accessed": s.LastAccessed,
	}
}

type Store struct {
	codecs     []securecookie.Codec
	options    *sessions.Options
	token      TokenInterface
	collection *mongo.Collection
	ctx        context.Context
}

func (s *Store) Get(r *http.Request, name string) (gSession *sessions.Session, err error) {
	return sessions.GetRegistry(r).Get(s, name)
}

func (s *Store) GetAndTouch(r *http.Request, w http.ResponseWriter, name string) (gSession *sessions.Session, err error) {

	gSession, err = s.Get(r, name)
	if err != nil {
		return nil, err
	}

	if gSession.IsNew {
		return gSession, err
	}

	// ensure access time is update to time.Now().UTC() in upsert
	delete(gSession.Values, lastAccessedField)

	err = s.Save(r, w, gSession)
	if err != nil {
		return nil, err
	}

	return gSession, err
}

func (s *Store) New(r *http.Request, name string) (gSession *sessions.Session, err error) {

	gSession = sessions.NewSession(s, name)
	gSession.Options = s.options
	gSession.IsNew = true

	val, errToken := s.token.GetToken(r, name)
	if errToken != nil {
		return gSession, err
	}

	err = securecookie.DecodeMulti(name, val, &gSession.ID, s.codecs...)
	if err != nil {
		return gSession, err
	}

	err = s.load(gSession)
	if err != nil {
		return gSession, err
	}

	gSession.IsNew = false

	return gSession, err
}

func (s *Store) Save(r *http.Request, w http.ResponseWriter, gSession *sessions.Session) (err error) {

	if gSession.Options.MaxAge < 0 {

		// Validate session ID
		objectID, err := primitive.ObjectIDFromHex(gSession.ID)
		if err != nil {
			return err
		}

		_, err = s.collection.DeleteOne(s.ctx, bson.M{"_id": objectID.String()}, options.Delete())
		if err != nil {
			return err
		}

		s.token.SetToken(w, gSession.Name(), "", gSession.Options)
		return nil
	}

	if gSession.ID == "" {
		gSession.ID = primitive.NewObjectID().Hex()
	}

	err = s.upsert(gSession)
	if err != nil {
		return err
	}

	var encoded string

	encoded, err = securecookie.EncodeMulti(gSession.Name(), gSession.ID, s.codecs...)
	if err != nil {
		return err
	}

	s.token.SetToken(w, gSession.Name(), encoded, gSession.Options)

	return err
}

func (s *Store) load(gSession *sessions.Session) error {

	// Validate session ID
	objectID, err := primitive.ObjectIDFromHex(gSession.ID)
	if err != nil {
		return err
	}

	result := s.collection.FindOne(s.ctx, bson.M{"_id": objectID.String()}, options.FindOne())
	if result.Err() != nil {
		return result.Err()
	}

	var mSession SessionRow
	err = result.Decode(&mSession)
	if err != nil {
		return err
	}

	err = securecookie.DecodeMulti(gSession.Name(), mSession.Data, &gSession.Values, s.codecs...)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) upsert(gSession *sessions.Session) error {

	// Validate session ID
	objectID, err := primitive.ObjectIDFromHex(gSession.ID)
	if err != nil {
		return err
	}

	// Get access time
	var accessed time.Time

	if val, ok := gSession.Values[lastAccessedField]; ok {

		accessed, ok = val.(time.Time)
		if !ok {
			return ErrInvalidLastAccessedTime
		}
	} else {
		accessed = time.Now().UTC()
	}

	// Encode data
	encoded, err := securecookie.EncodeMulti(gSession.Name(), gSession.Values, s.codecs...)
	if err != nil {
		return err
	}

	// Upsert into MongoDB
	sess := SessionRow{
		ID:           objectID.String(),
		Data:         encoded,
		LastAccessed: accessed,
	}

	_, err = s.collection.UpdateOne(s.ctx, bson.M{"_id": sess.ID}, sess.bson(), options.Update().SetUpsert(true))
	return err
}
