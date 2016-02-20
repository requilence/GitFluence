package main

type UserStat struct {
	Lines       LinesStat
	LinesPerExt map[string]*LinesStat
	Email       string
	CommitID    string
	CommitDays  int
	Username    string
}

type LinesStat struct {
	LastMonth  int
	Last3Month int
	Last6Month int
	LastYear   int
	Total      int
}

func (l *LinesStat) Append(lines LinesStat) {
	l.LastMonth += lines.LastMonth
	l.Last3Month += lines.Last3Month
	l.Last6Month += lines.Last6Month
	l.LastYear += lines.LastYear
	l.Total += lines.Total
}

type FileStat struct {
	TotalLines int
	Users      map[string]*UserStat
}

type RepoStat struct {
	Lines LinesStat
	Users map[string]*UserStat
	Files map[string]*FileStat
}

type RepoConfig struct {
	URL string
}
type Repo struct {
	Token string
	Host  string
	Owner string
	Name  string
	Stat  RepoStat
}
