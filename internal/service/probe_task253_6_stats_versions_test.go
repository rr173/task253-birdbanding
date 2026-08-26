package service
import "testing"
func TestStatsIncludesPersistedVersions(t *testing.T) {
	svc:=New(newTestDB(t)); ind,_,err:=svc.ResolveIndividual("AB1234","duck"); if err!=nil {t.Fatal(err)}
	if _,err:=svc.CreateVersion(ind.ID,"v1"); err!=nil {t.Fatal(err)}
	stats,err:=svc.Stats(); if err!=nil {t.Fatal(err)}
	if stats["versions"] != 1 { t.Fatalf("统计应包含版本数量，实际 %d",stats["versions"]) }
}
