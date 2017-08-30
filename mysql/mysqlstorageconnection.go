package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/FeiniuBus/capgo"
	_ "github.com/go-sql-driver/mysql"
)

// MySqlStorageConnection ...
type MySqlStorageConnection struct {
	Options *cap.CapOptions
}

// NewStorageConnection ...
func NewStorageConnection(options *cap.CapOptions) cap.IStorageConnection {
	connection := &MySqlStorageConnection{}
	connection.Options = options
	return connection
}

// OpenDbConnection ...
func (connection MySqlStorageConnection) OpenDbConnection() (*sql.DB, error) {
	connectionString, err := connection.Options.GetConnectionString()
	if err != nil {
		return nil, err
	}
	conn, err := sql.Open("mysql", connectionString)

	if err != nil {
		return nil, err
	}
	return conn, nil
}

// BeginTransaction ...
func (connection MySqlStorageConnection) BeginTransaction(dbConnection *sql.DB) (*sql.Tx, error) {
	options := &sql.TxOptions{Isolation: sql.LevelReadCommitted}
	transaction, err := dbConnection.BeginTx(context.Background(), options)
	if err != nil {
		return nil, err
	}
	return transaction, nil
}

// CreateTransaction ...
func (connection *MySqlStorageConnection) CreateTransaction() (cap.IStorageTransaction, error) {
	transaction, err := NewStorageTransaction(connection.Options)
	if err != nil {
		return nil, err
	}
	return transaction, nil
}

// FetchNextMessage ...
func (connection *MySqlStorageConnection) FetchNextMessage() (cap.IFetchedMessage, error) {
	conn, err := connection.OpenDbConnection()
	if err != nil {
		return nil, err
	}

	transaction, err := connection.BeginTransaction(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}

	statement := "SELECT `MessageId`,`MessageType` FROM `cap.queue` LIMIT 1 FOR UPDATE;DELETE FROM `cap.queue` LIMIT 1;"

	row, err := transaction.Query(statement)
	defer row.Close()
	if err != nil {
		conn.Close()
		return nil, err
	}

	var messageID int
	var messageType int

	if row.Next() == true {
		row.Scan(&messageID, &messageType)
	} else {
		conn.Close()
		return nil, nil
	}

	if messageID == 0 {
		return nil, nil
	}

	fetchedMessage := NewFetchedMessage(messageID, messageType, conn, transaction)

	return fetchedMessage, nil
}

// GetFailedPublishedMessages ...
func (connection *MySqlStorageConnection) GetFailedPublishedMessages() ([]*cap.CapPublishedMessage, error) {
	statement := "SELECT `Id`, CONVERT(UNIX_TIMESTAMP(`Added`),SIGNED) AS Added, `Content`, CONVERT(UNIX_TIMESTAMP(`ExpiresAt`),SIGNED) AS ExpiresAt, CONVERT(UNIX_TIMESTAMP(`LastWarnedTime`),SIGNED) AS LastWarnedTime,  `MessageId`, `Name`, `Retries`, `StatusName`, `TransactionId` FROM `cap.published` WHERE `StatusName` = 'Failed';"
	conn, err := connection.OpenDbConnection()
	defer conn.Close()
	if err != nil {
		return nil, err
	}

	returnValue := make([]*cap.CapPublishedMessage, 0)

	rows, err := conn.Query(statement)

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		item := &cap.CapPublishedMessage{}
		err = rows.Scan(&item.Id, &item.Added, &item.Content, &item.ExpiresAt, &item.LastWarnedTime, &item.MessageId, &item.Name, &item.Retries, &item.StatusName, &item.TransactionId)
		if err != nil {
			return nil, err
		}
		returnValue = append(returnValue, item)
	}
	return returnValue, nil
}

// GetFailedReceivedMessages ...
func (connection *MySqlStorageConnection) GetFailedReceivedMessages() ([]*cap.CapReceivedMessage, error) {
	statement := "SELECT `Id`, CONVERT(UNIX_TIMESTAMP(`Added`),SIGNED) AS Added, `Content`, CONVERT(UNIX_TIMESTAMP(`ExpiresAt`),SIGNED) AS ExpiresAt, `Group`, CONVERT(UNIX_TIMESTAMP(`LastWarnedTime`),SIGNED) AS LastWarnedTime, `MessageId`, `Name`, `Retries`, `StatusName`, `TransactionId` FROM `cap.received` WHERE `StatusName` = 'Failed';"
	conn, err := connection.OpenDbConnection()
	defer conn.Close()
	if err != nil {
		return nil, err
	}

	returnValue := make([]*cap.CapReceivedMessage, 0)

	rows, err := conn.Query(statement)

	if err != nil {
		return nil, err
	}

	for rows.Next() {
		item := &cap.CapReceivedMessage{}
		err = rows.Scan(&item.Id, &item.Added, &item.Content, &item.ExpiresAt, &item.Group, &item.LastWarnedTime, &item.MessageId, &item.Name, &item.Retries, &item.StatusName, &item.TransactionId)
		if err != nil {
			return nil, err
		}
		returnValue = append(returnValue, item)
	}
	return returnValue, nil
}

// GetNextPublishedMessageToBeEnqueued ...
func (connection *MySqlStorageConnection) GetNextPublishedMessageToBeEnqueued() (*cap.CapPublishedMessage, error) {
	statement := "SELECT `Id`, CONVERT(UNIX_TIMESTAMP(`Added`),SIGNED) AS Added, `Content`, CONVERT(UNIX_TIMESTAMP(`ExpiresAt`),SIGNED) AS ExpiresAt, CONVERT(UNIX_TIMESTAMP(`LastWarnedTime`),SIGNED) AS LastWarnedTime, `MessageId`, `Name`, `Retries`, `StatusName`, `TransactionId` FROM `cap.published` WHERE `StatusName` = 'Scheduled' LIMIT 1;"
	conn, err := connection.OpenDbConnection()

	if err != nil {
		return nil, err
	}

	if conn == nil {
		return nil, cap.NewCapError("Database connection is nil.")
	}

	defer conn.Close()

	rows, err := conn.Query(statement)
	if err != nil {
		return nil, err
	}
	message := &cap.CapPublishedMessage{}
	if rows.Next() {
		rows.Scan(&message.Id, &message.Added, &message.Content, &message.ExpiresAt, &message.LastWarnedTime, &message.MessageId, &message.Name, &message.Retries, &message.StatusName, &message.TransactionId)
	}

	return message, nil
}

// GetNextReceviedMessageToBeEnqueued ..
func (connection *MySqlStorageConnection) GetNextReceviedMessageToBeEnqueued() (*cap.CapReceivedMessage, error) {
	statement := "SELECT `Id`, CONVERT(UNIX_TIMESTAMP(`Added`),SIGNED) AS Added, `Content`, CONVERT(UNIX_TIMESTAMP(`ExpiresAt`),SIGNED) AS ExpiresAt, `Group`, CONVERT(UNIX_TIMESTAMP(`LastWarnedTime`),SIGNED) AS LastWarnedTime, `MessageId`, `Name`, `Retries`, `StatusName`, `TransactionId` FROM `cap.received` WHERE `StatusName` = 'Scheduled' LIMIT 1;"
	conn, err := connection.OpenDbConnection()
	defer conn.Close()
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query(statement)
	if err != nil {
		return nil, err
	}
	message := &cap.CapReceivedMessage{}

	if rows.Next() {
		rows.Scan(&message.Id, &message.Added, &message.Content, &message.ExpiresAt, &message.Group, &message.LastWarnedTime, &message.MessageId, &message.Name, &message.Retries, &message.StatusName, &message.TransactionId)
	}

	return message, nil
}

// GetPublishedMessage ...
func (connection *MySqlStorageConnection) GetPublishedMessage(id int) (*cap.CapPublishedMessage, error) {
	statement := "SELECT `Id`, CONVERT(UNIX_TIMESTAMP(`Added`),SIGNED) AS Added, `Content`, CONVERT(UNIX_TIMESTAMP(`ExpiresAt`),SIGNED) AS ExpiresAt, CONVERT(UNIX_TIMESTAMP(`LastWarnedTime`),SIGNED) AS LastWarnedTime, `MessageId`, `Name`, `Retries`, `StatusName`, `TransactionId` FROM `cap.published` WHERE `Id`=?;"
	conn, err := connection.OpenDbConnection()
	defer conn.Close()
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query(statement, id)
	if err != nil {
		return nil, err
	}
	message := &cap.CapPublishedMessage{}

	if rows.Next() {
		rows.Scan(&message.Id, &message.Added, &message.Content, &message.ExpiresAt, &message.LastWarnedTime, &message.MessageId, &message.Name, &message.Retries, &message.StatusName, &message.TransactionId)
	}

	return message, nil
}

// GetReceivedMessage ...
func (connection *MySqlStorageConnection) GetReceivedMessage(id int) (*cap.CapReceivedMessage, error) {
	statement := "SELECT `Id`, CONVERT(UNIX_TIMESTAMP(`Added`),SIGNED) AS Added, `Content`, CONVERT(UNIX_TIMESTAMP(`ExpiresAt`),SIGNED) AS ExpiresAt, `Group`, CONVERT(UNIX_TIMESTAMP(`LastWarnedTime`),SIGNED) AS LastWarnedTime, `MessageId`, `Name`, `Retries`, `StatusName`, `TransactionId` FROM `cap.received` WHERE `Id`=?;"
	conn, err := connection.OpenDbConnection()
	defer conn.Close()
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query(statement, id)
	if err != nil {
		return nil, err
	}
	message := &cap.CapReceivedMessage{}

	if rows.Next() {
		rows.Scan(&message.Id, &message.Added, &message.Content, &message.ExpiresAt, &message.Group, &message.LastWarnedTime, &message.MessageId, &message.Name, &message.Retries, &message.StatusName, &message.TransactionId)
	}

	return message, nil
}

// StoreReceivedMessage ...
func (connection *MySqlStorageConnection) StoreReceivedMessage(message *cap.CapReceivedMessage) error {
	statement := "INSERT INTO `cap.received`(`Name`,`Group`,`Content`,`Retries`,`Added`,`ExpiresAt`,`StatusName`,`MessageId`,`TransactionId`)"
	statement += " VALUES(?,?,?,?,?,?,?,?,?);"
	conn, err := connection.OpenDbConnection()
	defer conn.Close()
	if err != nil {
		return err
	}

	feiniuMessage := cap.FeiniuBusMessage{
		MetaData: cap.FeiniuBusMessageMetaData{},
	}
	err = json.Unmarshal([]byte(message.Content), feiniuMessage)
	if err != nil {
		if connection.Options.LogFunction != nil {
			connection.Options.LogFunction(err.Error(), cap.NewInfomationLevel())
		}
		return err
	}

	result, err := conn.Exec(statement, message.Name, message.Group, feiniuMessage.Content, message.Retries, time.Now(), nil, message.StatusName, feiniuMessage.MetaData.MessageID, feiniuMessage.MetaData.TransactionID)
	if err != nil {
		return err
	}
	rowAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowAffected == int64(0) {
		return cap.NewCapError("StoreReceivedMessage : Database execution should affect 1 row but affected 0 row actually.")
	}
	return nil
}
