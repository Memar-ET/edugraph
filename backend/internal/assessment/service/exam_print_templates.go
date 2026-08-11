package service

import "html/template"

// Both templates below are plain print-ready HTML, not server-rendered
// PDFs -- Capability 2.2 (PRD Section 4/Module B "Mode B: Print & Paper
// Exam") asks for a "printable PDF exam sheet". Generating an actual PDF
// server-side would mean adding a new third-party Go PDF library, which
// this change deliberately avoids: every modern browser already renders
// HTML to a correctly paginated, correctly fonted PDF via its native
// "Print > Save as PDF", so returning well-styled HTML and letting the
// browser do that conversion is simpler, has zero new dependencies, and
// produces an equal or better result than most from-scratch Go PDF
// generation would. The frontend's "Print exam" / "Print answer key"
// buttons (ExamReviewPage.tsx) open these in a new tab and call
// window.print() automatically.
//
// html/template (not text/template) is used deliberately: question text
// and options come from parsed teacher-uploaded documents and are not
// trusted input, so auto-escaping matters here the same way it would for
// any other user-supplied content rendered as HTML.

const printCSS = `
  * { box-sizing: border-box; }
  body { font-family: Georgia, 'Times New Roman', serif; color: #111; max-width: 800px; margin: 0 auto; padding: 32px; line-height: 1.5; }
  .no-print { }
  @media print { .no-print { display: none !important; } body { padding: 0; } }
  .print-bar { text-align: right; margin-bottom: 16px; }
  .print-bar button { font: inherit; padding: 8px 16px; cursor: pointer; }
  header.exam-header { border-bottom: 2px solid #111; padding-bottom: 12px; margin-bottom: 20px; }
  header.exam-header h1 { font-size: 1.4rem; margin: 0 0 4px; }
  header.exam-header .meta { font-size: 0.95rem; color: #333; }
  .fill-fields { display: flex; flex-wrap: wrap; gap: 24px; margin: 16px 0 24px; font-size: 0.95rem; }
  .fill-fields span { border-bottom: 1px solid #111; display: inline-block; min-width: 160px; }
  .instructions { font-size: 0.9rem; font-style: italic; margin-bottom: 24px; }
  .question { margin-bottom: 22px; page-break-inside: avoid; }
  .question .q-head { font-weight: bold; }
  .question .marks { font-weight: normal; font-style: italic; color: #444; }
  .options { list-style: none; padding-left: 1.5em; margin: 8px 0 0; }
  .options li { margin-bottom: 4px; }
  .answer-space { border-bottom: 1px solid #999; height: 1.6em; margin-top: 10px; }
  table.bubble-sheet { width: 100%; border-collapse: collapse; margin-top: 12px; }
  table.bubble-sheet th, table.bubble-sheet td { border: 1px solid #999; padding: 6px 10px; text-align: center; font-size: 0.9rem; }
  table.bubble-sheet td.bubble { width: 28px; }
  .bubble-mark { display: inline-block; width: 16px; height: 16px; border-radius: 50%; border: 1.5px solid #111; }
  .bubble-mark.filled { background: #111; }
  .non-mcq-note { color: #555; font-style: italic; }
`

const printBar = `
<div class="print-bar no-print">
  <button onclick="window.print()">Print / Save as PDF</button>
</div>
`

var printExamTemplate = template.Must(template.New("printExam").Parse(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>{{.Exam.Title}} - Exam Sheet</title>
<style>` + printCSS + `</style>
</head>
<body>
` + printBar + `
<header class="exam-header">
  <h1>{{.Exam.Title}}</h1>
  <div class="meta">{{.Exam.SubjectCode}} &middot; Grade {{.Exam.GradeLevel}} &middot; {{.Exam.ExamScope}} &middot; {{.Exam.AcademicYear}} &middot; Total: {{.Exam.TotalMarks}} marks</div>
</header>

<div class="fill-fields">
  <div>School: <span>&nbsp;</span></div>
  <div>Student name: <span>&nbsp;</span></div>
  <div>Section: <span>&nbsp;</span></div>
  <div>Date: <span>&nbsp;</span></div>
</div>

<p class="instructions">Answer all questions in the space provided. Read each question carefully before answering.</p>

{{range .Questions}}
<div class="question">
  <div class="q-head">{{.SequenceNumber}}.{{if .PartLabel}} ({{.PartLabel}}){{end}} {{.QuestionText}} <span class="marks">[{{.Marks}} mark{{if ne .Marks 1}}s{{end}}]</span></div>
  {{if .Options}}
  <ol class="options" type="A">
    {{range .Options}}<li>({{.Letter}}) {{.Text}}</li>
    {{end}}
  </ol>
  {{else}}
  <div class="answer-space"></div>
  <div class="answer-space"></div>
  {{end}}
</div>
{{end}}

</body>
</html>
`))

var answerKeyTemplate = template.Must(template.New("answerKey").Parse(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>{{.Exam.Title}} - Answer Key</title>
<style>` + printCSS + `</style>
</head>
<body>
` + printBar + `
<header class="exam-header">
  <h1>{{.Exam.Title}} &mdash; Answer Key</h1>
  <div class="meta">{{.Exam.SubjectCode}} &middot; Grade {{.Exam.GradeLevel}} &middot; {{.Exam.ExamScope}} &middot; {{.Exam.AcademicYear}} &middot; Teacher reference only &mdash; do not distribute to students</div>
</header>

<table class="bubble-sheet">
  <thead>
    <tr>
      <th>Q#</th>
      <th>Marks</th>
      <th>A</th><th>B</th><th>C</th><th>D</th>
      <th>Notes</th>
    </tr>
  </thead>
  <tbody>
    {{range .Rows}}
    <tr>
      <td>{{.SequenceNumber}}</td>
      <td>{{.Marks}}</td>
      {{if eq .QuestionType "mcq"}}
        {{$correct := .CorrectOption}}
        <td class="bubble"><span class="bubble-mark{{if eq $correct "A"}} filled{{end}}"></span></td>
        <td class="bubble"><span class="bubble-mark{{if eq $correct "B"}} filled{{end}}"></span></td>
        <td class="bubble"><span class="bubble-mark{{if eq $correct "C"}} filled{{end}}"></span></td>
        <td class="bubble"><span class="bubble-mark{{if eq $correct "D"}} filled{{end}}"></span></td>
        <td></td>
      {{else}}
        <td colspan="4"></td>
        <td class="non-mcq-note">graded manually ({{.QuestionType}})</td>
      {{end}}
    </tr>
    {{end}}
  </tbody>
</table>

</body>
</html>
`))
