// Package 'kvlite' provides a Key Value interface upon SQLite.
package kvlite

import (
	"github.com/mattn/go-sqlite3"
	"database/sql"
	"encoding/gob"
	"bytes"
	"fmt"
	"strings"
	"sync"
)

type Store struct {
		key		[]byte
		filePath	string
		mutex		sync.RWMutex
		encoder		*gob.Encoder
		buffer		*bytes.Buffer
		dbCon		*sql.DB
}

const (
	_none = (1 << iota)
	_encrypt
	_sort
	_revsort
	_reserved
)

// Checks to see if table name is reserved or invalid.
func chkTable(table *string, flags int) (err error) {
	for _, ch := range *table {
		switch ch {
			case 0x3b:
				fallthrough
			case 0x22:
				fallthrough
			case 0x27:
				fallthrough
			case 0x26:
				fallthrough
			case 0x28:
				return fmt.Errorf("Invalid characters in table name: '%s'", *table)
		}
	}
	
	if flags & _reserved > 0 { return }
	if *table == "KVLite" { return fmt.Errorf("Sorry, %s is a reserved name.", *table) }
	return		
}

// Stores value in Store datastore.
func (s *Store) Set(table string, key string, val interface{}) (err error) {
	return s.set(table, key, val, 0)
}

// Writes encrypted value to Store datastore.
func (s *Store) CryptSet(table, key string, val interface{}) (err error) {
	return s.set(table, key, val, _encrypt)
}

// Internal function to write to SQLite.
func (s *Store) set(table string, key string, val interface{}, flags int) (err error) {

	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	//val = convertPrimitives(val)
	
	var (
		eFlag int
		encBytes []byte
	)
	
	// Encode data.
	switch v := val.(type) {
		case []byte:
			encBytes = v
		default:
			s.buffer.Reset()
			err = s.encoder.Encode(val)
			if err != nil { return err }
			encBytes = s.buffer.Bytes()
	}
			
	err = chkTable(&table, flags)
	if err != nil { return err }

	if flags & _encrypt != 0 { 
		encBytes = encrypt(encBytes, s.key) 
		eFlag = 1
	}

	_, err = s.dbCon.Exec("CREATE TABLE IF NOT EXISTS '" + table + "' (key TEXT PRIMARY KEY, value BLOB, e int)")
	if err != nil { return err }
	
	s.dbCon.Exec("DELETE FROM '" + table + "' WHERE key COLLATE nocase = ?;", key);
	_, err = s.dbCon.Exec("INSERT OR REPLACE INTO '"+table+"'(key,value,e) VALUES(?, ?, ?);", key, encBytes, eFlag)
	if err != nil { return err }

	return
}

// Unset/remove key in table specified.
func (s *Store) Unset(table string, key string) (error) {
	return s.unset(table, key, 0)
}

func (s *Store) unset(table string, key string, flags int) (err error) {

	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	err = chkTable(&table, flags)
	if err != nil { return err }

	if _, err := s.dbCon.Exec("DELETE FROM '" + table + "' WHERE key COLLATE nocase = ?;", key); err != nil {
		if strings.Contains(err.Error(), "no such table") == true {
			return nil
		}
		if err != nil { return err }
	}
	return
}

// Truncates a table in Store datastore.
func (s *Store) Truncate(table string) (error) {
	return s.truncate(table, 0)
}

func (s *Store) truncate(table string, flags int) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	
	err = chkTable(&table, flags)
	if err != nil { return err }
	
	if _, err := s.dbCon.Exec("DROP TABLE '" + table + "';"); err != nil {
		if strings.Contains(err.Error(), "no such table") == true {	return err }
	}
	return nil
}

// Retreive a value at key in table specified.
func (s *Store) Get(table string, key string, output interface{}) (found bool, err error) {
	
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var eFlag int
	var data []byte
	
	err = chkTable(&table, _reserved)
	if err != nil { return false, err }
	
	err = s.dbCon.QueryRow("SELECT value FROM '"+table+"' WHERE key COLLATE nocase = ?", key).Scan(&data)

	switch {
	case err == sql.ErrNoRows:
		return false, err
	case err != nil:
		if strings.Contains(err.Error(), "no such table") == true {
			return false, nil
		} else { return false, err }
	default:
		err = s.dbCon.QueryRow("SELECT e FROM '"+table+"' WHERE key COLLATE nocase = ?;", key).Scan(&eFlag)
		if err != nil { return false, err }
		if eFlag != 0 { data = decrypt(data, s.key) }
	}
	
	switch o := output.(type) {
		case *[]byte:
			*o = append(*o, data[0:]...)
		default:
			var dec *gob.Decoder
			dec = gob.NewDecoder(bytes.NewReader(data))
			if dec != nil { return true, dec.Decode(output) } 
	}
	
	
	return true, nil
}

// List all tables, if filter specified only tables that match filter.
func (s *Store) ListTables(filter string) (cList []string, err error) {
	
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var rows *sql.Rows

	if filter == "" {
		rows, err = s.dbCon.Query("SELECT name FROM sqlite_master WHERE type='table';")
		if err != nil { return nil, err }
	} else {
		rows, err = s.dbCon.Query("SELECT name FROM sqlite_master WHERE type='table' and name like ?;", filter)
		if err != nil { return nil, err }
	}

	defer rows.Close()

	for rows.Next() {
		var table string
		err = rows.Scan(&table)
		if err != nil { return nil, err }

		if table != "KVLite" {
			cList = append(cList, table)
		} 
	}

	err = rows.Err()
	if err != nil { return nil, err }

	return
}

// List all keys in table, only those matching filter if specified.
func (s *Store) CountKeys(table string, filter string) (count uint32, err error) {
	
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var rows *sql.Rows
	
	err = chkTable(&table, _reserved)
	if err != nil { return 0, err }
	
	if filter != "" {
		rows, err = s.dbCon.Query("SELECT COUNT(key) FROM '" + table + "' where key like ?;", filter)
	} else {
		rows, err = s.dbCon.Query("SELECT COUNT(key) FROM '" + table + "';")
	}

	// Prevent table does not exist errors.
	if err != nil {
		if strings.Contains(err.Error(), "no such table") == true {
			return 0, nil
		} else {
			return 0, err
		}
	}

	defer rows.Close()

	for rows.Next() {
		err = rows.Scan(&count)
		if err != nil { return 0, err }
	}
	err = rows.Err()
	if err != nil { return 0, err }

	return
}


// List all keys in table, only those matching filter if specified.
func (s *Store) ListKeys(table string, filter string) (keyList []string, err error) {
	
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	var rows *sql.Rows
	
	err = chkTable(&table, _reserved)
	if err != nil { return nil, err }
	
	if filter != "" {
		rows, err = s.dbCon.Query("SELECT key FROM '" + table + "' where key like ?;", filter)
	} else {
		rows, err = s.dbCon.Query("SELECT key FROM '" + table + "';")
	}

	// Prevent table does not exist errors.
	if err != nil {
		if strings.Contains(err.Error(), "no such table") == true {
			return nil, nil
		} else {
			return nil, err
		}
	}

	defer rows.Close()

	for rows.Next() {
		var key string
		err = rows.Scan(&key)
		if err != nil { return nil, err }
		keyList = append(keyList, key)
	}
	err = rows.Err()
	if err != nil { return nil, err }

	return
}

// Close Store.
func (s *Store) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	return s.dbCon.Close()
}

// Manually override encryption key used with CryptSet.
func (s *Store) CryptKey(key []byte) {
	s.key = key
}

var _Store_DRIVER string

func init() {
	sql.Register(_Store_DRIVER, &sqlite3.SQLiteDriver{})
}

// Open or Creates a new KvStore, if autoCrypt is set to true will use auto-created encryption key.
func Open(filePath string, padlock...[]byte) (*Store, error) {
	if filePath == "" { return nil, fmt.Errorf("kvlite: Missing filename parameter.")}
	if len(padlock) == 0 {
		return open(filePath, nil, 0)
	} else {
		for i, pad := range padlock {
			if i == 0 { continue }
			padlock[0] = append(padlock[0], pad[0:]...)
			padlock[i] = nil
		}
		return open(filePath, padlock[0], 0)
	}
}

func open(filePath string, padlock []byte, flags int) (openStore *Store, err error) {
	dbCon, err := sql.Open(_Store_DRIVER, filePath)
	if err != nil { return nil, err }
	
	var buff bytes.Buffer

	openStore = &Store{
		dbCon:		dbCon,
		filePath:	filePath,
		buffer: 	&buff,
		encoder: gob.NewEncoder(&buff),
	}

	if err = dbCon.Ping(); err != nil { 
		dbCon.Close()
		fmt.Errorf("%s: %s", filePath, err.Error()) 
		return nil, err
	}
	
	_, err = dbCon.Exec("PRAGMA case_sensitive_like=OFF;")
	if err != nil { 
		dbCon.Close()
		return nil, err 
	}
	
	if flags & _reserved == 0 {
		err = openStore.dbunlocker(padlock)
		if err != nil { return nil, err }
	}
	
	return
}
