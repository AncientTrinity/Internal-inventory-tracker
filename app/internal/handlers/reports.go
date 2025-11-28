// filename: internal/handlers/reports.go
package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ReportsHandler struct {
	DB *sql.DB
}

func NewReportsHandler(db *sql.DB) *ReportsHandler {
	return &ReportsHandler{DB: db}
}

// ReportFilter represents the filter criteria for reports
type ReportFilter struct {
	StartDate  time.Time `json:"start_date"`
	EndDate    time.Time `json:"end_date"`
	AssetType  *string   `json:"asset_type,omitempty"`
	TicketType *string   `json:"ticket_type,omitempty"`
	UserID     *int64    `json:"user_id,omitempty"`
	Priority   *string   `json:"priority,omitempty"`
}

// GET /api/v1/reports/analytics
func (h *ReportsHandler) GetAnalytics(w http.ResponseWriter, r *http.Request) {
	var filter ReportFilter

	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		http.Error(w, "Invalid filter data", http.StatusBadRequest)
		return
	}

	// Validate date range
	if filter.EndDate.Before(filter.StartDate) {
		http.Error(w, "End date cannot be before start date", http.StatusBadRequest)
		return
	}

	// Get comprehensive analytics data
	analytics, err := h.getComprehensiveAnalytics(filter)
	if err != nil {
		fmt.Printf("Error getting analytics: %v\n", err)
		http.Error(w, "Failed to generate analytics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

// GET /api/v1/reports/export/csv
func (h *ReportsHandler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	var filter ReportFilter

	if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
		http.Error(w, "Invalid filter data", http.StatusBadRequest)
		return
	}

	// Generate CSV data
	csvData, err := h.generateCSVReport(filter)
	if err != nil {
		http.Error(w, "Failed to generate CSV report", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=report.csv")
	w.Write([]byte(csvData))
}

// GET /api/v1/reports/types
func (h *ReportsHandler) GetReportTypes(w http.ResponseWriter, r *http.Request) {
	reportTypes := []string{
		"ticket_analytics",
		"asset_utilization", 
		"user_activity",
		"system_metrics",
		"performance_report",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"report_types": reportTypes,
	})
}

// Helper method to get comprehensive analytics
func (h *ReportsHandler) getComprehensiveAnalytics(filter ReportFilter) (map[string]interface{}, error) {
	analytics := make(map[string]interface{})

	// Get ticket statistics
	ticketStats, err := h.getTicketStatistics(filter)
	if err != nil {
		return nil, err
	}
	analytics["ticket_stats"] = ticketStats

	// Get asset statistics
	assetStats, err := h.getAssetStatistics(filter)
	if err != nil {
		return nil, err
	}
	analytics["asset_stats"] = assetStats

	// Get ticket trends
	ticketTrends, err := h.getTicketTrends(filter)
	if err != nil {
		return nil, err
	}
	analytics["ticket_trends"] = ticketTrends

	// Get asset utilization
	assetUtilization, err := h.getAssetUtilization(filter)
	if err != nil {
		return nil, err
	}
	analytics["asset_utilization"] = assetUtilization

	// Get user activity
	userActivity, err := h.getUserActivity(filter)
	if err != nil {
		return nil, err
	}
	analytics["user_activity"] = userActivity

	return analytics, nil
}

// Get ticket statistics
func (h *ReportsHandler) getTicketStatistics(filter ReportFilter) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'open' THEN 1 END) as open,
			COUNT(CASE WHEN status = 'received' THEN 1 END) as received,
			COUNT(CASE WHEN status = 'in_progress' THEN 1 END) as in_progress,
			COUNT(CASE WHEN status = 'resolved' THEN 1 END) as resolved,
			COUNT(CASE WHEN status = 'closed' THEN 1 END) as closed,
			COUNT(CASE WHEN priority = 'critical' THEN 1 END) as critical,
			COUNT(CASE WHEN priority = 'high' THEN 1 END) as high,
			COUNT(CASE WHEN priority = 'normal' THEN 1 END) as normal,
			COUNT(CASE WHEN priority = 'low' THEN 1 END) as low,
			COALESCE(AVG(EXTRACT(EPOCH FROM (closed_at - created_at))/3600), 0) as avg_resolution_hours
		FROM tickets 
		WHERE created_at BETWEEN $1 AND $2
	`

	var stats struct {
		Total               int     `json:"total"`
		Open                int     `json:"open"`
		Received            int     `json:"received"`
		InProgress          int     `json:"in_progress"`
		Resolved            int     `json:"resolved"`
		Closed              int     `json:"closed"`
		Critical            int     `json:"critical"`
		High                int     `json:"high"`
		Normal              int     `json:"normal"`
		Low                 int     `json:"low"`
		AvgResolutionHours  float64 `json:"avg_resolution_hours"`
	}

	err := h.DB.QueryRow(query, filter.StartDate, filter.EndDate).Scan(
		&stats.Total, &stats.Open, &stats.Received, &stats.InProgress,
		&stats.Resolved, &stats.Closed, &stats.Critical, &stats.High,
		&stats.Normal, &stats.Low, &stats.AvgResolutionHours,
	)

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"total":                  stats.Total,
		"open":                   stats.Open,
		"received":               stats.Received,
		"in_progress":            stats.InProgress,
		"resolved":               stats.Resolved,
		"closed":                 stats.Closed,
		"critical":               stats.Critical,
		"high":                   stats.High,
		"normal":                 stats.Normal,
		"low":                    stats.Low,
		"avg_resolution_hours":   stats.AvgResolutionHours,
	}, nil
}

// Get asset statistics

func (h *ReportsHandler) getAssetStatistics(filter ReportFilter) (map[string]interface{}, error) {
    query := `
        SELECT 
            COUNT(*) as total_assets,
            COUNT(CASE WHEN status = 'IN_USE' THEN 1 END) as in_use,
            COUNT(CASE WHEN status = 'IN_STORAGE' THEN 1 END) as in_storage,
            COUNT(CASE WHEN status = 'REPAIR' THEN 1 END) as in_repair,
            COUNT(CASE WHEN status = 'RETIRED' THEN 1 END) as retired,
            COUNT(CASE WHEN next_service_date <= NOW() THEN 1 END) as needs_service,
            COUNT(DISTINCT asset_type) as asset_types_count
        FROM assets
    `

    var stats struct {
        TotalAssets     int `json:"total_assets"`
        InUse           int `json:"in_use"`
        InStorage       int `json:"in_storage"`
        InRepair        int `json:"in_repair"`
        Retired         int `json:"retired"`
        NeedsService    int `json:"needs_service"`
        AssetTypesCount int `json:"asset_types_count"`
    }

    err := h.DB.QueryRow(query).Scan(
        &stats.TotalAssets, &stats.InUse, &stats.InStorage, &stats.InRepair,
        &stats.Retired, &stats.NeedsService, &stats.AssetTypesCount,
    )

    if err != nil {
        return nil, err
    }

    // Debug output
    fmt.Printf("🔍 ASSET STATS (Using correct status values):\n")
    fmt.Printf("🔍   - Total: %d\n", stats.TotalAssets)
    fmt.Printf("🔍   - IN_USE: %d\n", stats.InUse)
    fmt.Printf("🔍   - IN_STORAGE: %d\n", stats.InStorage)
    fmt.Printf("🔍   - REPAIR: %d\n", stats.InRepair)
    fmt.Printf("🔍   - RETIRED: %d\n", stats.Retired)
    fmt.Printf("🔍   - Utilization Rate: %.1f%%\n", float64(stats.InUse)/float64(stats.TotalAssets)*100)

    return map[string]interface{}{
        "total_assets":      stats.TotalAssets,
        "in_use":            stats.InUse,
        "in_storage":        stats.InStorage,
        "in_repair":         stats.InRepair,
        "retired":           stats.Retired,
        "needs_service":     stats.NeedsService,
        "asset_types_count": stats.AssetTypesCount,
    }, nil
}

// Get ticket trends (last 30 days)
func (h *ReportsHandler) getTicketTrends(filter ReportFilter) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			DATE(created_at) as date,
			COUNT(*) as count,
			COUNT(CASE WHEN status = 'closed' THEN 1 END) as resolved_count
		FROM tickets 
		WHERE created_at BETWEEN $1 AND $2
		GROUP BY DATE(created_at)
		ORDER BY date
	`

	rows, err := h.DB.Query(query, filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var trends []map[string]interface{}
	for rows.Next() {
		var date time.Time
		var count, resolvedCount int

		err := rows.Scan(&date, &count, &resolvedCount)
		if err != nil {
			return nil, err
		}

		trends = append(trends, map[string]interface{}{
			"date":            date.Format("2006-01-02"),
			"count":           count,
			"resolved_count":  resolvedCount,
		})
	}

	return trends, nil
}

// Get asset utilization by type
func (h *ReportsHandler) getAssetUtilization(filter ReportFilter) ([]map[string]interface{}, error) {
    query := `
        SELECT 
            asset_type,
            COUNT(*) as total,
            COUNT(CASE WHEN status = 'IN_USE' THEN 1 END) as in_use,
            COUNT(CASE WHEN status = 'IN_STORAGE' THEN 1 END) as available,
            COUNT(CASE WHEN status = 'REPAIR' THEN 1 END) as in_repair
        FROM assets
        GROUP BY asset_type
        ORDER BY total DESC
    `

    rows, err := h.DB.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var utilization []map[string]interface{}
    for rows.Next() {
        var assetType string
        var total, inUse, available, inRepair int

        err := rows.Scan(&assetType, &total, &inUse, &available, &inRepair)
        if err != nil {
            return nil, err
        }

        utilizationRate := 0.0
        if total > 0 {
            utilizationRate = float64(inUse) / float64(total) * 100
        }

        utilization = append(utilization, map[string]interface{}{
            "asset_type":        assetType,
            "total":             total,
            "in_use":            inUse,
            "available":         available,
            "in_repair":         inRepair,
            "utilization_rate":  utilizationRate,
        })
    }

    return utilization, nil
}

// Get user activity statistics
func (h *ReportsHandler) getUserActivity(filter ReportFilter) ([]map[string]interface{}, error) {
	query := `
		SELECT 
			u.id,
			u.username,
			u.full_name,
			u.role_id,
			COUNT(t.id) as tickets_created,
			COUNT(DISTINCT a.id) as assets_assigned,
			COUNT(CASE WHEN t.status = 'closed' THEN 1 END) as tickets_resolved
		FROM users u
		LEFT JOIN tickets t ON u.id = t.created_by AND t.created_at BETWEEN $1 AND $2
		LEFT JOIN assets a ON u.id = a.in_use_by
		GROUP BY u.id, u.username, u.full_name, u.role_id
		ORDER BY tickets_created DESC
	`

	rows, err := h.DB.Query(query, filter.StartDate, filter.EndDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activity []map[string]interface{}
	for rows.Next() {
		var userID, roleID int64
		var username, fullName string
		var ticketsCreated, assetsAssigned, ticketsResolved int

		err := rows.Scan(&userID, &username, &fullName, &roleID, 
			&ticketsCreated, &assetsAssigned, &ticketsResolved)
		if err != nil {
			return nil, err
		}

		activity = append(activity, map[string]interface{}{
			"user_id":          userID,
			"username":         username,
			"full_name":        fullName,
			"role_id":          roleID,
			"tickets_created":  ticketsCreated,
			"assets_assigned":  assetsAssigned,
			"tickets_resolved": ticketsResolved,
		})
	}

	return activity, nil
}

// Generate CSV report
func (h *ReportsHandler) generateCSVReport(filter ReportFilter) (string, error) {
	analytics, err := h.getComprehensiveAnalytics(filter)
	if err != nil {
		return "", err
	}

	// Enhanced CSV with better structure
	csv := "Internal Inventory Tracker Report\n"
	csv += fmt.Sprintf("Generated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	csv += fmt.Sprintf("Period: %s to %s\n\n", 
		filter.StartDate.Format("2006-01-02"), 
		filter.EndDate.Format("2006-01-02"))

	// Ticket Statistics Section
	csv += "TICKET STATISTICS\n"
	csv += "Metric,Value\n"
	ticketStats := analytics["ticket_stats"].(map[string]interface{})
	for key, value := range ticketStats {
		csv += fmt.Sprintf("%s,%v\n", h.formatKey(key), value)
	}
	csv += "\n"

	// Asset Statistics Section  
	csv += "ASSET STATISTICS\n"
	csv += "Metric,Value\n"
	assetStats := analytics["asset_stats"].(map[string]interface{})
	for key, value := range assetStats {
		csv += fmt.Sprintf("%s,%v\n", h.formatKey(key), value)
	}
	csv += "\n"

	// Ticket Trends Section
	csv += "TICKET TRENDS (Last 30 Days)\n"
	csv += "Date,Total Tickets,Resolved Tickets\n"
	ticketTrends := analytics["ticket_trends"].([]map[string]interface{})
	for _, trend := range ticketTrends {
		csv += fmt.Sprintf("%s,%v,%v\n", 
			trend["date"], trend["count"], trend["resolved_count"])
	}

	return csv, nil
}

func (h *ReportsHandler) formatKey(key string) string {
	// Convert snake_case to Title Case
	words := strings.Split(key, "_")
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}