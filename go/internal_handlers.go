package main

import (
	"net/http"
)

// このAPIをインスタンス内から一定間隔で叩かせることで、椅子とライドをマッチングさせる
// func internalGetMatching(w http.ResponseWriter, r *http.Request) {
// 	ctx := r.Context()
// 	// MEMO: 一旦最も待たせているリクエストに適当な空いている椅子マッチさせる実装とする。おそらくもっといい方法があるはず…
// 	ride := &Ride{}
// 	if err := db.GetContext(ctx, ride, `SELECT * FROM rides WHERE chair_id IS NULL ORDER BY created_at LIMIT 1`); err != nil {
// 		if errors.Is(err, sql.ErrNoRows) {
// 			w.WriteHeader(http.StatusNoContent)
// 			return
// 		}
// 		writeError(w, http.StatusInternalServerError, err)
// 		return
// 	}

// 	matched := &Chair{}
// 	empty := false
// 	for i := 0; i < 10; i++ {
// 		if err := db.GetContext(ctx, matched, "SELECT * FROM chairs INNER JOIN (SELECT id FROM chairs WHERE is_active = TRUE ORDER BY RAND() LIMIT 1) AS tmp ON chairs.id = tmp.id LIMIT 1"); err != nil {
// 			if errors.Is(err, sql.ErrNoRows) {
// 				w.WriteHeader(http.StatusNoContent)
// 				return
// 			}
// 			writeError(w, http.StatusInternalServerError, err)
// 		}

// 		if err := db.GetContext(ctx, &empty, "SELECT COUNT(*) = 0 FROM (SELECT COUNT(chair_sent_at) = 6 AS completed FROM ride_statuses WHERE ride_id IN (SELECT id FROM rides WHERE chair_id = ?) GROUP BY ride_id) is_completed WHERE completed = FALSE", matched.ID); err != nil {
// 			writeError(w, http.StatusInternalServerError, err)
// 			return
// 		}
// 		if empty {
// 			break
// 		}
// 	}
// 	if !empty {
// 		w.WriteHeader(http.StatusNoContent)
// 		return
// 	}

// 	if _, err := db.ExecContext(ctx, "UPDATE rides SET chair_id = ? WHERE id = ?", matched.ID, ride.ID); err != nil {
// 		writeError(w, http.StatusInternalServerError, err)
// 		return
// 	}

// 	w.WriteHeader(http.StatusNoContent)
// }
func internalGetMatching(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Step 1: 未割り当てのライドをすべて取得
	rides := []Ride{}
	if err := db.SelectContext(ctx, &rides, `SELECT * FROM rides WHERE chair_id IS NULL ORDER BY created_at ASC`); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(rides) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Step 2: 有効な椅子（最新の状態と位置情報を含む）を取得
	chairs := []ChairWithLatLon{}
	query := `
WITH chair_latest_location AS (
    SELECT *
    FROM (
        SELECT chair_locations.*, ROW_NUMBER() OVER (PARTITION BY chair_id ORDER BY created_at DESC) AS rn
        FROM chair_locations
    ) c
    WHERE c.rn = 1
),
chair_latest_status AS (
    SELECT *
    FROM (
        SELECT rides.*, ride_statuses.status AS ride_status, ROW_NUMBER() OVER (PARTITION BY chair_id ORDER BY ride_statuses.created_at DESC) AS rn
        FROM rides
        INNER JOIN ride_statuses ON rides.id = ride_statuses.ride_id AND ride_statuses.chair_sent_at IS NOT NULL
    ) r
    WHERE r.rn = 1
)
SELECT
    chairs.*, chair_latest_location.latitude, chair_latest_location.longitude
FROM chairs
LEFT JOIN chair_latest_status ON chairs.id = chair_latest_status.chair_id
LEFT JOIN chair_latest_location ON chairs.id = chair_latest_location.chair_id
WHERE
    (chair_latest_status.ride_status = 'COMPLETED' OR chair_latest_status.ride_status IS NULL)
    AND chairs.is_active
`
	if err := db.SelectContext(ctx, &chairs, query); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if len(chairs) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	for _, ride := range rides {
		minDistance := 400
		var minChair *ChairWithLatLon
		var minChairIdx int
		for idx, chair := range chairs {
			distance := calculateDistance(chair.Latitude, chair.Longitude, ride.PickupLatitude, ride.PickupLongitude)
			if distance < minDistance {
				minDistance = distance
				minChair = &chair
				minChairIdx = idx
			}
		}
		if minChair != nil {
			// トランザクションを使用して割り当て処理を安全に行う
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}

			_, err = tx.ExecContext(ctx, "UPDATE rides SET chair_id = ? WHERE id = ?", minChair.ID, ride.ID)
			if err != nil {
				tx.Rollback()
				writeError(w, http.StatusInternalServerError, err)
				return
			}

			if err := tx.Commit(); err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}

			// 割り当てた椅子をリストから削除
			chairs = append(chairs[:minChairIdx], chairs[minChairIdx+1:]...)
		}
	}

	// 処理完了
	w.WriteHeader(http.StatusNoContent)
}
