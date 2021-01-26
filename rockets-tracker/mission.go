package main

import (
	"sort"

	"github.com/fanaticscripter/EggContractor/api"
	"github.com/fanaticscripter/EggContractor/util"
)

type mission struct {
	Ship                api.MissionInfo_Spaceship    `json:"ship"`
	ShipName            string                       `json:"shipName"`
	ShipIconPath        string                       `json:"shipIconPath"`
	Status              api.MissionInfo_Status       `json:"status"`
	StatusDisplay       string                       `json:"statusDisplay"`
	DurationType        api.MissionInfo_DurationType `json:"durationType"`
	DurationTypeDisplay string                       `json:"durationTypeDisplay"`
	DurationSeconds     float64                      `json:"durationSeconds"`
	DurationDisplay     string                       `json:"durationDisplay"`
	Capacity            uint32                       `json:"capacity"`
	StartTimestamp      float64                      `json:"startTimestamp"`
	ReturnTimestamp     float64                      `json:"returnTimestamp"`
}

type missionStats struct {
	Ships    []*shipMissionStats `json:"ships"`
	shipsMap map[api.MissionInfo_Spaceship]*shipMissionStats
}

type shipMissionStats struct {
	Ship         api.MissionInfo_Spaceship `json:"ship"`
	ShipName     string                    `json:"shipName"`
	ShipIconPath string                    `json:"shipIconPath"`
	Types        []*shipTypeMissionStats   `json:"types"`
	Count        uint32                    `json:"count"`
	typesMap     map[api.MissionInfo_DurationType]*shipTypeMissionStats
}

type shipTypeMissionStats struct {
	DurationType        api.MissionInfo_DurationType `json:"durationType"`
	DurationTypeDisplay string                       `json:"durationTypeDisplay"`
	DurationSeconds     float64                      `json:"durationSeconds"`
	DurationDisplay     string                       `json:"durationDisplay"`
	Capacity            uint32                       `json:"capacity"`
	Fuels               []*fuel                      `json:"fuels"`
	Count               uint32                       `json:"count"`
}

type fuel struct {
	Egg           api.EggType `json:"egg"`
	EggDisplay    string      `json:"eggDisplay"`
	EggIconPath   string      `json:"eggIconPath"`
	Amount        float64     `json:"amount"`
	AmountDisplay string      `json:"amountDisplay"`
}

type unlockProgress struct {
	NextShip         api.MissionInfo_Spaceship `json:"nextShip"`
	NextShipName     string                    `json:"nextShipName"`
	NextShipIconPath string                    `json:"nextShipIconPath"`
	LaunchesRequired uint32                    `json:"launchesRequired"`
	LaunchesDone     uint32                    `json:"launchesDone"`
}

type launchLog struct {
	Dates []*launchLogDate `json:"dates"`
}

type launchLogDate struct {
	Date     string     `json:"date"`
	Missions []*mission `json:"missions"`
}

func newMission(m *api.MissionInfo) *mission {
	startTimestamp := m.StartTimeDerived
	returnTimestamp := 0.0
	if startTimestamp > 0 {
		returnTimestamp = startTimestamp + m.DurationSeconds
	}
	return &mission{
		Ship:                m.Ship,
		ShipName:            m.Ship.Name(),
		ShipIconPath:        shipIconPath(m.Ship),
		Status:              m.Status,
		StatusDisplay:       m.Status.Display(),
		DurationType:        m.DurationType,
		DurationTypeDisplay: m.DurationType.Display(),
		DurationSeconds:     m.DurationSeconds,
		DurationDisplay:     util.FormatDurationWhole(util.DoubleToDuration(m.DurationSeconds)),
		Capacity:            m.Capacity,
		StartTimestamp:      startTimestamp,
		ReturnTimestamp:     returnTimestamp,
	}
}

func generateStatsFromMissionArchive(archive []*api.MissionInfo) (*missionStats, *unlockProgress) {
	shipsMap := make(map[api.MissionInfo_Spaceship]*shipMissionStats)
	for _, m := range archive {
		ship, ok := shipsMap[m.Ship]
		if !ok {
			ship = &shipMissionStats{
				Ship:         m.Ship,
				ShipName:     m.Ship.Name(),
				ShipIconPath: shipIconPath(m.Ship),
				typesMap:     make(map[api.MissionInfo_DurationType]*shipTypeMissionStats),
			}
			shipsMap[m.Ship] = ship
		}
		ship.Count++
		typ, ok := ship.typesMap[m.DurationType]
		if !ok {
			var fuels []*fuel
			for _, f := range m.Fuel {
				fuels = append(fuels, &fuel{
					Egg:           f.Egg,
					EggDisplay:    f.Egg.Display(),
					EggIconPath:   "egginc/" + f.Egg.IconFilename(),
					Amount:        f.Amount,
					AmountDisplay: util.NumfmtWhole(f.Amount),
				})
			}
			sort.Slice(fuels, func(i, j int) bool {
				return fuels[i].Egg < fuels[j].Egg
			})
			typ = &shipTypeMissionStats{
				DurationType:        m.DurationType,
				DurationTypeDisplay: m.DurationType.Display(),
				DurationSeconds:     m.DurationSeconds,
				DurationDisplay:     util.FormatDurationWhole(util.DoubleToDuration(m.DurationSeconds)),
				Capacity:            m.Capacity,
				Fuels:               fuels,
			}
			ship.typesMap[m.DurationType] = typ
		}
		typ.Count++
	}
	ships := make([]*shipMissionStats, 0)
	for _, ship := range shipsMap {
		types := make([]*shipTypeMissionStats, 0)
		for _, typ := range ship.typesMap {
			types = append(types, typ)
		}
		sort.Slice(types, func(i, j int) bool {
			di := types[i].DurationType
			dj := types[j].DurationType
			// Assume di != dj
			switch {
			case di == api.MissionInfo_TUTORIAL:
				return true
			case dj == api.MissionInfo_EPIC:
				return true
			case di == api.MissionInfo_SHORT && dj == api.MissionInfo_LONG:
				return true
			default:
				return false
			}
		})
		ship.Types = types
		ships = append(ships, ship)
	}
	sort.Slice(ships, func(i, j int) bool {
		return ships[i].Ship < ships[j].Ship
	})
	stats := &missionStats{
		Ships:    ships,
		shipsMap: shipsMap,
	}

	if len(ships) == 0 {
		return stats, &unlockProgress{
			NextShip:         api.MissionInfo_CHICKEN_ONE,
			NextShipName:     api.MissionInfo_CHICKEN_ONE.Name(),
			NextShipIconPath: shipIconPath(api.MissionInfo_CHICKEN_ONE),
			LaunchesRequired: 0,
			LaunchesDone:     0,
		}
	}
	lastShip := ships[len(ships)-1].Ship
	if lastShip == api.MissionInfo_HENERPRISE {
		return stats, nil
	}
	nextShip := lastShip + 1
	progress := &unlockProgress{
		NextShip:         nextShip,
		NextShipName:     nextShip.Name(),
		NextShipIconPath: shipIconPath(nextShip),
		LaunchesRequired: shipRequiredLaunchesToUnlock(nextShip),
		LaunchesDone:     ships[len(ships)-1].Count,
	}

	return stats, progress
}

func generateLaunchLogFromMissionArchive(archive []*api.MissionInfo) *launchLog {
	sort.SliceStable(archive, func(i, j int) bool {
		return archive[i].StartTime().After(archive[j].StartTime())
	})
	date2missions := make(map[string][]*mission)
	for _, m := range archive {
		if m.StartTime().IsZero() {
			continue
		}
		date := util.FormatDate(m.StartTime())
		date2missions[date] = append(date2missions[date], newMission(m))
	}
	dates := make([]string, 0)
	for d := range date2missions {
		dates = append(dates, d)
	}
	sort.SliceStable(dates, func(i, j int) bool {
		return dates[i] > dates[j]
	})
	log := &launchLog{}
	for _, d := range dates {
		log.Dates = append(log.Dates, &launchLogDate{
			Date:     d,
			Missions: date2missions[d],
		})
	}
	return log
}

func shipIconPath(ship api.MissionInfo_Spaceship) string {
	return "egginc/" + ship.IconFilename()
}

func shipRequiredLaunchesToUnlock(ship api.MissionInfo_Spaceship) uint32 {
	switch ship {
	case api.MissionInfo_CHICKEN_ONE:
		return 0
	case api.MissionInfo_CHICKEN_NINE:
		return 4
	case api.MissionInfo_CHICKEN_HEAVY:
		return 6
	case api.MissionInfo_BCR:
		return 12
	case api.MissionInfo_MILLENIUM_CHICKEN:
		return 15
	case api.MissionInfo_CORELLIHEN_CORVETTE:
		return 18
	case api.MissionInfo_GALEGGTICA:
		return 21
	case api.MissionInfo_CHICKFIANT:
		return 24
	case api.MissionInfo_VOYEGGER:
		return 27
	case api.MissionInfo_HENERPRISE:
		return 30
	}
	return 0
}
