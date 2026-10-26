package admin

import "time"

type DailyStat struct {
	Day      string `json:"day"`
	Searches int    `json:"searches"`
	NewUsers int    `json:"new_users"`
}

type TopUser struct {
	Username string `json:"username"`
	Searches int    `json:"searches"`
}

type StatsResponse struct {
	Days     int         `json:"days"`
	Daily    []DailyStat `json:"daily"`
	TopUsers []TopUser   `json:"top_users"`
}

// GetStats 返回最近 N 天的搜索/新增用户趋势与搜索排行。
func GetStats(days int) (*StatsResponse, error) {
	if days < 1 || days > 90 {
		days = 14
	}
	start := time.Now().UTC().AddDate(0, 0, -(days - 1))
	startDay := start.Format("2006-01-02")
	startTS := startDay + "T00:00:00Z"

	searchCounts := map[string]int{}
	rows, err := DB.Query(
		`SELECT substr(created_at, 1, 10) AS day, COUNT(*) FROM search_logs
		 WHERE created_at >= ? GROUP BY day`, startTS,
	)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var day string
		var count int
		if err := rows.Scan(&day, &count); err != nil {
			rows.Close()
			return nil, err
		}
		searchCounts[day] = count
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	userCounts := map[string]int{}
	rows, err = DB.Query(
		`SELECT substr(created_at, 1, 10) AS day, COUNT(*) FROM users
		 WHERE created_at >= ? GROUP BY day`, startTS,
	)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var day string
		var count int
		if err := rows.Scan(&day, &count); err != nil {
			rows.Close()
			return nil, err
		}
		userCounts[day] = count
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	daily := make([]DailyStat, 0, days)
	for i := 0; i < days; i++ {
		day := start.AddDate(0, 0, i).Format("2006-01-02")
		daily = append(daily, DailyStat{
			Day:      day,
			Searches: searchCounts[day],
			NewUsers: userCounts[day],
		})
	}

	topUsers := []TopUser{}
	rows, err = DB.Query(
		`SELECT username, COUNT(*) AS c FROM search_logs
		 WHERE user_id > 0 AND created_at >= ?
		 GROUP BY user_id, username ORDER BY c DESC LIMIT 8`, startTS,
	)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var item TopUser
		if err := rows.Scan(&item.Username, &item.Searches); err != nil {
			rows.Close()
			return nil, err
		}
		topUsers = append(topUsers, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	return &StatsResponse{Days: days, Daily: daily, TopUsers: topUsers}, nil
}
