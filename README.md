# kvlite
--
    import "github.com/cmcoffee/go-kvlite"

Package 'kvlite' provides a Key Value interface upon SQLite.

## Usage

```go
var ErrBadPadlock = errors.New("kvlite: Invalid padlock provided, unable to open database.")
```
ErrBadPadlock is returned if kvlite.Open is used with incorrect padlock set on
database.

```go
var ErrBadPass = errors.New("kvlite: Invalid passphrase provided, unable to remove lock!")
```
ErrBadPass is returned if an Unlock is attempted with the incorrect passphrase.

```go
var ErrNotUnlocked = errors.New("kvlite: Cannot apply new lock on top of existing lock, must remove old lock first.")
```
ErrLocked is returned if a new Lock is attempted on a database that is currently
locked.

#### func  Lock

```go
func Lock(filepath, passphrase string, padlock []byte) (err error)
```
Sets a lock on Store database, requires a passphrase (for unlocking in future)
and padlock when opening database in future.

#### func  Unlock

```go
func Unlock(filepath, passphrase string) (err error)
```
Removes lock on Store database, strips the requirement for padlock for opening
database, requires passphrase set on initial lock.

#### type Store

```go
type Store struct {
}
```


#### func  Open

```go
func Open(filePath string, padlock ...[]byte) (*Store, error)
```
Open or Creates a new KvStore, if autoCrypt is set to true will use auto-created
encryption key.

#### func (*Store) Close

```go
func (s *Store) Close() error
```
Close Store.

#### func (*Store) CountKeys

```go
func (s *Store) CountKeys(table, filter string) (count int, err error)
```
Return a count of all keys in specified table.

#### func (*Store) CryptKey

```go
func (s *Store) CryptKey(key []byte)
```
Manually override encryption key used with CryptSet.

#### func (*Store) CryptSet

```go
func (s *Store) CryptSet(table, key string, val interface{}) (err error)
```
Writes encrypted value to Store datastore.

#### func (*Store) Get

```go
func (s *Store) Get(table string, key string, output interface{}) (found bool, err error)
```
Retreive a value at key in table specified.

#### func (*Store) ListKeys

```go
func (s *Store) ListKeys(table string, filter string) (keyList []string, err error)
```
List all keys in table, only those matching filter if specified.

#### func (*Store) ListTables

```go
func (s *Store) ListTables(filter string) (cList []string, err error)
```
List all tables, if filter specified only tables that match filter.

#### func (*Store) Set

```go
func (s *Store) Set(table string, key string, val interface{}) (err error)
```
Stores value in Store datastore.

#### func (*Store) Truncate

```go
func (s *Store) Truncate(table string) error
```
Truncates a table in Store datastore.

#### func (*Store) Unset

```go
func (s *Store) Unset(table string, key string) error
```
Unset/remove key in table specified.
