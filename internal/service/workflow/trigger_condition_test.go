package workflow

import (
	"testing"

	"github.com/thecoretg/ticketbot/models"
)

// A trigger's own condition gates its lane: a failing or erroring condition records the trigger
// step and starts nothing; a passing one walks as before.
func TestTriggerConditionGatesTheLane(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	yes, no, bad := trig("yes", 0), trig("no", 100), trig("bad", 200)
	yes.Condition = "status/id = 10"
	no.Condition = "status/id = 99"
	bad.Condition = "summary =" // fails to compile at evaluation, the same error path an if takes
	w := graph([]models.Node{
		yes, no, bad,
		act("n1", notify(models.ChannelWebexRoom, rid(1))), act("n2", notify(models.ChannelWebexRoom, rid(2))), act("n3", notify(models.ChannelWebexRoom, rid(3))),
	}, edge("yes", models.PortOut, "n1"), edge("no", models.PortOut, "n2"), edge("bad", models.PortOut, "n3"))
	res := exec(t, cw, w, Input{})

	eq(t, "recipients", recipients(res), []int{1})
	eq(t, "path", path(res), []string{"yes", "n1", "no", "bad"})
	if s := step(res, "yes"); s.Matched == nil || !*s.Matched || s.Port != models.PortOut {
		t.Errorf("passing trigger = %+v", s)
	}
	if s := step(res, "no"); s.Matched == nil || *s.Matched || s.Port != "" {
		t.Errorf("failing trigger should record matched=false and no port: %+v", s)
	}
	if s := step(res, "bad"); s.Err == nil || s.Port != "" {
		t.Errorf("erroring trigger should record the error and not walk: %+v", s)
	}
}
