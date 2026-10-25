package queue

const MaxPlaceSize = 5

type Entry struct {
	UserID   int64
	Username string
	SortKey  int64
}

func Places(entries []Entry) [][]Entry {
	var places [][]Entry

	for i, e := range entries {
		if i == 0 || e.SortKey != entries[i-1].SortKey {
			places = append(places, nil)
		}

		places[len(places)-1] = append(places[len(places)-1], e)
	}

	return places
}

func PlaceOf(places [][]Entry, userID int64) int {
	for i, place := range places {
		for _, e := range place {
			if e.UserID == userID {
				return i + 1
			}
		}
	}

	return 0
}
