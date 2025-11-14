package models

import (
	"database/sql"
	"errors"
	"time"
)

type TicketComment struct {
	ID        int64     `json:"id"`
	TicketID  int64     `json:"ticket_id"`
	AuthorID  *int64    `json:"author_id"`
	Comment   string    `json:"comment"`
	IsInternal bool     `json:"is_internal"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	
	// Joined fields
	Author *User `json:"author,omitempty"`
}

type TicketCommentModel struct {
	DB *sql.DB
}

func NewTicketCommentModel(db *sql.DB) *TicketCommentModel {
	return &TicketCommentModel{DB: db}
}

// Insert a new ticket comment
func (m *TicketCommentModel) Insert(comment *TicketComment) error {
	query := `
		INSERT INTO ticket_comments (ticket_id, author_id, comment, is_internal)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	
	err := m.DB.QueryRow(
		query,
		comment.TicketID,
		comment.AuthorID,
		comment.Comment,
		comment.IsInternal,
	).Scan(&comment.ID, &comment.CreatedAt, &comment.UpdatedAt)
	
	return err
}

// Get comments for a ticket
func (m *TicketCommentModel) GetByTicketID(ticketID int64, showInternal bool) ([]TicketComment, error) {
	query := `
		SELECT 
			tc.id, tc.ticket_id, tc.author_id, tc.comment, tc.is_internal,
			tc.created_at, tc.updated_at,
			u.username, u.full_name
		FROM ticket_comments tc
		LEFT JOIN users u ON tc.author_id = u.id
		WHERE tc.ticket_id = $1
	`
	
	args := []interface{}{ticketID}
	
	if !showInternal {
		query += " AND tc.is_internal = false"
	}
	
	query += " ORDER BY tc.created_at ASC"
	
	rows, err := m.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var comments []TicketComment
	for rows.Next() {
		var comment TicketComment
		var authorID sql.NullInt64
		var username, fullName sql.NullString
		
		err := rows.Scan(
			&comment.ID,
			&comment.TicketID,
			&authorID,
			&comment.Comment,
			&comment.IsInternal,
			&comment.CreatedAt,
			&comment.UpdatedAt,
			&username, &fullName,
		)
		if err != nil {
			return nil, err
		}
		
		if authorID.Valid {
			comment.AuthorID = &authorID.Int64
			comment.Author = &User{
				Username: username.String,
				FullName: fullName.String,
			}
		}
		
		comments = append(comments, comment)
	}
	
	return comments, nil
}

// Get comment by ID
func (m *TicketCommentModel) GetByID(id int64) (*TicketComment, error) {
	var comment TicketComment
	var authorID sql.NullInt64
	var username, fullName sql.NullString
	
	query := `
		SELECT 
			tc.id, tc.ticket_id, tc.author_id, tc.comment, tc.is_internal,
			tc.created_at, tc.updated_at,
			u.username, u.full_name
		FROM ticket_comments tc
		LEFT JOIN users u ON tc.author_id = u.id
		WHERE tc.id = $1
	`
	
	err := m.DB.QueryRow(query, id).Scan(
		&comment.ID,
		&comment.TicketID,
		&authorID,
		&comment.Comment,
		&comment.IsInternal,
		&comment.CreatedAt,
		&comment.UpdatedAt,
		&username, &fullName,
	)
	
	if err == sql.ErrNoRows {
		return nil, errors.New("comment not found")
	} else if err != nil {
		return nil, err
	}
	
	if authorID.Valid {
		comment.AuthorID = &authorID.Int64
		comment.Author = &User{
			Username: username.String,
			FullName: fullName.String,
		}
	}
	
	return &comment, nil
}

// Update a comment
func (m *TicketCommentModel) Update(comment *TicketComment) error {
	query := `
		UPDATE ticket_comments 
		SET comment = $1, is_internal = $2, updated_at = NOW()
		WHERE id = $3
		RETURNING updated_at
	`
	
	err := m.DB.QueryRow(
		query,
		comment.Comment,
		comment.IsInternal,
		comment.ID,
	).Scan(&comment.UpdatedAt)
	
	if err == sql.ErrNoRows {
		return errors.New("comment not found")
	}
	return err
}

// Delete a comment
func (m *TicketCommentModel) Delete(id int64) error {
	result, err := m.DB.Exec("DELETE FROM ticket_comments WHERE id = $1", id)
	if err != nil {
		return err
	}
	
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return errors.New("comment not found")
	}
	return nil
}