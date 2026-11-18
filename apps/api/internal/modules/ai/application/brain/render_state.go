package brain

import (
	"strings"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// Source handles are attributes, never prose. Absolute persistence timestamps
// are deliberately omitted: Brain receives local time/relative ages instead.
func renderState(w c.WorkingContext, safety c.SafetySignal) string {
	var out strings.Builder
	out.WriteString("<state>\n<session id=\"" + w.Session.SessionID.String() + "\">")
	pending := struct {
		Slot string `json:"slot"`
		Text string `json:"text"`
	}{}
	if w.Session.PendingQuestion != nil {
		pending.Slot = w.Session.PendingQuestion.Slot
		pending.Text = w.Session.PendingQuestion.Text
	}
	out.WriteString(data("session_data", struct {
		Mode          c.SessionMode `json:"mode"`
		TurnCount     int           `json:"turn_count"`
		LastObjective *c.Objective  `json:"last_objective,omitempty"`
		Consecutive   int           `json:"consecutive_questions"`
		Pending       any           `json:"pending_question"`
		Summary       string        `json:"summary,omitempty"`
		Emotions      []string      `json:"emotion_trajectory,omitempty"`
		Resumed       bool          `json:"resumed"`
	}{w.Session.Mode, w.Session.TurnCount, w.Session.LastObjective, w.Session.ConsecutiveQuestions, pending, w.Session.Summary, w.Session.EmotionTrajectory, w.Session.Resumed}))
	out.WriteString("</session>\n")
	if t := w.FocusThread; t != nil {
		out.WriteString("<thread id=\"" + t.ID.String() + "\">")
		out.WriteString(data("thread_data", struct {
			Topic   string                   `json:"topic"`
			Phase   c.ThreadPhase            `json:"phase"`
			Goal    *string                  `json:"goal,omitempty"`
			History []c.ProtocolHistoryBrief `json:"protocol_history,omitempty"`
		}{t.Topic, t.Phase, t.Goal, nil}))
		for _, s := range t.Slots {
			ids := []string{}
			for _, id := range s.Evidence {
				ids = append(ids, id.String())
			}
			out.WriteString("<slot name=\"" + escaped(s.Slot) + "\" epistemic=\"" + escaped(string(s.Status)) + "\" evidence=\"" + strings.Join(ids, ",") + "\" declined=\"" + boolText(s.Declined) + "\">" + escaped(s.Value) + "</slot>\n")
		}
		for _, h := range t.Hypotheses {
			out.WriteString("<hypothesis id=\"" + h.ID.String() + "\" status=\"" + escaped(string(h.Status)) + "\">")
			out.WriteString(data("hypothesis_data", struct {
				Text       string            `json:"text"`
				Confidence c.ConfidenceClass `json:"confidence"`
				Support    int               `json:"support"`
				Contradict int               `json:"contradict"`
				Voiced     bool              `json:"voiced"`
			}{h.Text, h.Confidence, h.Support, h.Contradict, h.Voiced}))
			out.WriteString("</hypothesis>\n")
		}
		out.WriteString("</thread>\n")
	}
	for _, t := range w.OtherThreads {
		out.WriteString("<other_thread id=\"" + t.ID.String() + "\" phase=\"" + escaped(string(t.Phase)) + "\">" + escaped(t.Topic) + "</other_thread>\n")
	}
	if p := w.Protocol; p != nil {
		out.WriteString("<protocol instance_id=\"" + p.InstanceID.String() + "\">")
		// InstanceID has already been rendered as an opaque handle. All other
		// fields are domain data (catalog/step ids are controlled, non-UUID ids).
		out.WriteString(data("protocol_data", struct {
			ProtocolID      string             `json:"protocol_id"`
			Version         string             `json:"version"`
			Status          c.ProtocolStatus   `json:"status"`
			StepID          string             `json:"step_id"`
			StepGoal        string             `json:"step_goal"`
			ExpectedInput   string             `json:"expected_input"`
			Attempts        int                `json:"attempts"`
			StuckCount      int                `json:"stuck_count"`
			LastUserSummary string             `json:"last_user_summary"`
			AllowedActions  []c.ProtocolAction `json:"allowed_actions"`
		}{p.ProtocolID, p.Version, p.Status, p.StepID, p.StepGoal, p.ExpectedInput, p.Attempts, p.StuckCount, p.LastUserSummary, p.AllowedActions}))
		out.WriteString("</protocol>\n")
	}
	out.WriteString(data("prefs", w.Prefs))
	out.WriteString(data("risk", w.Risk))
	out.WriteString(data("safety", safety))
	for _, l := range w.OpenLoops {
		ref := ""
		if l.Ref != nil {
			ref = l.Ref.String()
		}
		out.WriteString("<open_loop id=\"" + l.ID.String() + "\" ref=\"" + ref + "\" kind=\"" + escaped(l.Kind) + "\">" + escaped(l.Text) + "</open_loop>\n")
	}
	out.WriteString(data("info_needs", w.InfoNeeds))
	out.WriteString(data("resumption", w.Resumption))
	if f := w.Formulation; f != nil {
		out.WriteString("<formulation thread_id=\"" + f.ThreadID.String() + "\">")
		out.WriteString(data("formulation_data", struct {
			Version   int      `json:"version"`
			Paragraph string   `json:"paragraph"`
			Gaps      []string `json:"gaps"`
		}{f.Version, f.Paragraph, f.Gaps}))
		out.WriteString("</formulation>\n")
	}
	out.WriteString(data("eligibility", w.Eligibility))
	out.WriteString("</state>\n")
	return out.String()
}
