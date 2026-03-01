 Plan: /bureau admin route with WYSIWYM content editor                                                                                                   
                                                        
 Context

 The campaign site's text content is currently hardcoded in templates/index.html. The user wants a password-protected /bureau admin route where the main
  text content can be edited live via a semantic rich-text (WYSIWYM) editor, without redeploying. The edited content must pass through Gin's
 html/template engine safely.

 ---
 Architecture

 Data flow

 data/.env           → BUREAU_PASSWORD at startup
 data/content.json   → live page content (in-memory + file)
 templates/index.html → uses {{.Var}} Go template variables
 templates/bureau_login.html + bureau_editor.html → admin UI
 src/main.go         → all backend logic

 /data/* is already gitignored (except .gitkeep), so both new data files are safe.

 Session management

 In-memory sync.Map keyed by random 32-byte base64 token stored in HTTP-only SameSite=Strict cookie bureau_session. No new dependencies.

 .env parsing

 Manual bufio.Scanner — no new dependency.

 ---
 Go content types (in main.go)

 type ContentItem struct {
     Label string `json:"label"`
     Text  string `json:"text"`
 }
 type Engagement struct {
     Num    string        `json:"num"`
     Title  string        `json:"title"`
     Breton string        `json:"breton"`
     Items  []ContentItem `json:"items"`
 }
 type Candidate struct {
     ID   int    `json:"id"`
     Name string `json:"name"`
     Info string `json:"info"`
 }
 type PageContent struct {
     PageTitle    string       `json:"page_title"`
     HeroTitle    string       `json:"hero_title"`
     Subtitle     string       `json:"subtitle"`
     IntroText    string       `json:"intro_text"`     // stores raw HTML
     Engagements  []Engagement `json:"engagements"`
     TeamIntro    string       `json:"team_intro"`
     Candidates   []Candidate  `json:"candidates"`
     FooterTitle  string       `json:"footer_title"`
     FooterText   string       `json:"footer_text"`
     HighlightBox string       `json:"highlight_box"`
 }

 Global guarded by sync.RWMutex: var (contentMu sync.RWMutex; siteContent PageContent).

 ---
 New routes

 GET  /bureau           → 302 to /bureau/edit (if cookie valid) or /bureau/login
 GET  /bureau/login     → render bureau_login.html
 POST /bureau/login     → validate BUREAU_PASSWORD → set cookie → 302 /bureau/edit
 GET  /bureau/edit      → render bureau_editor.html (protected)
 POST /bureau/save      → JSON body → write data/content.json + update in-memory (protected)
 GET  /bureau/logout    → delete session → 302 /bureau/login

 authMiddleware helper reads cookie, looks up sync.Map, aborts with 302 to login if missing.

 ---
 src/main.go changes

 1. Add imports: bufio, crypto/rand, encoding/base64, encoding/json, html/template, strings, sync
 2. Add loadEnv(path) — reads data/.env, calls os.Setenv
 3. Add loadContent(path) / saveContent(path, c) — JSON marshal/unmarshal
 4. Add session helpers: newSession(), isAuthenticated(c *gin.Context) bool
 5. Pass content to GET /: extract siteContent under RLock, build gin.H with template.HTML(content.IntroText) for the raw HTML field, all others as
 plain strings/slices
 6. Add all 6 /bureau/* handlers
 7. In POST /bureau/save: sanitize by stripping any {{ / }} sequences before storing (prevents template injection)
 8. Call loadEnv("../data/.env") and loadContent("../data/content.json") near top of main()

 ---
 templates/index.html changes

 Replace every hardcoded text block with a template variable. Key changes:

 ┌───────────────────────────────────────────────────┬─────────────────────────────────────────────────────────────────────────────────────┐
 │                      Current                      │                                       Becomes                                       │
 ├───────────────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────┤
 │ <h1>Élections municipales 2026</h1>               │ <h1>{{.PageTitle}}</h1>                                                             │
 ├───────────────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────┤
 │ <div class="hero-title">UNIS POUR LANVAUDAN</div> │ <div class="hero-title">{{.HeroTitle}}</div>                                        │
 ├───────────────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────┤
 │ <div class="subtitle">UNANIT...</div>             │ <div class="subtitle">{{.Subtitle}}</div>                                           │
 ├───────────────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────┤
 │ <div class="intro-text reveal"><p>...</p></div>   │ <div class="intro-text reveal">{{.IntroText}}</div>                                 │
 ├───────────────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────┤
 │ 6 hardcoded .engagement-card divs                 │ {{range .Engagements}}...<li><strong>{{.Label}} :</strong> {{.Text}}</li>...{{end}} │
 ├───────────────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────┤
 │ <p class="reveal">Une équipe...</p>               │ <p class="reveal">{{.TeamIntro}}</p>                                                │
 ├───────────────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────┤
 │ <tbody id="teamBody"> (JS-populated)              │ {{range .Candidates}}<tr class="team-row">...</tr>{{end}}                           │
 ├───────────────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────┤
 │ Footer <h3>NOTRE MÉTHODE...</h3>                  │ <h3>{{.FooterTitle}}</h3>                                                           │
 ├───────────────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────┤
 │ Footer <p>Nous ne promettons...</p>               │ <p>{{.FooterText}}</p>                                                              │
 ├───────────────────────────────────────────────────┼─────────────────────────────────────────────────────────────────────────────────────┤
 │ <div class="highlight-box">...</div>              │ <div class="highlight-box">{{.HighlightBox}}</div>                                  │
 └───────────────────────────────────────────────────┴─────────────────────────────────────────────────────────────────────────────────────┘

 Remove the entire const candidates = [...] JS block and teamBody.forEach(...) population code from the <script> section.

 ---
 data/content.json

 Initial content extracted verbatim from current index.html. intro_text value:
 <p><strong>Chères Lanvaudanaises, chers Lanvaudanais,</strong></p><p>Les 15 et 22 mars 2026, vous élirez votre conseil municipal. Nous avons constitué
 une liste citoyenne, apolitique et participative, composée de femmes et d'hommes attachés à notre identité rurale et à notre cadre de vie. Notre projet
  s'articule autour de six engagements majeurs pour l'avenir de notre commune.</p>

 ---
 data/.env

 BUREAU_PASSWORD=changeme
 (User must change this before first use.)

 ---
 Editor UI (bureau_editor.html)

 - Font/colors match the main site (Montserrat, CSS vars)
 - Fixed top bar: "Bureau · Unis pour Lanvaudan" + "Se déconnecter" button
 - Three tabs: Textes | Engagements | Équipe
 - Textes tab: <input> fields for PageTitle, HeroTitle, Subtitle, TeamIntro, FooterTitle, HighlightBox; <textarea> for FooterText; Quill editor for
 IntroText
 - Engagements tab: 6 collapsible cards, each with text inputs for title/breton and a dynamic item list (label + text per item, add/remove buttons)
 - Équipe tab: Table of candidates rows with editable name/info inputs, add/remove row buttons
 - Fixed bottom save bar with a "Enregistrer" button; JS collects all fields, fetch POST /bureau/save as JSON, shows success/error toast

 Quill.js (CDN, Snow theme)

 Used only for IntroText (the only true rich-text field). Toolbar restricted to ['bold', 'italic'] and [{ list: 'ordered' }, { list: 'bullet' }]. No
 colors, fonts, sizes — enforces semantic HTML only.

 Initialize: quill.root.innerHTML = data.intro_text
 Retrieve: quill.root.innerHTML (then strip <p><br></p> if empty)

 ---
 assets/sw.js change

 Add /bureau to the exclusion check:
 if (event.request.url.includes('/subscribe') || event.request.url.includes('/bureau')) {
     return;
 }

 ---
 Files modified/created

 ┌──────────────────────────────┬─────────────────────────────────────────────────────────────────────────────┐
 │             File             │                                   Action                                    │
 ├──────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
 │ src/main.go                  │ Major additions — content types, env/content loading, session, 6 new routes │
 ├──────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
 │ templates/index.html         │ Template variables replacing hardcoded content; remove JS candidate array   │
 ├──────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
 │ templates/bureau_login.html  │ New — login form                                                            │
 ├──────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
 │ templates/bureau_editor.html │ New — full admin editor                                                     │
 ├──────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
 │ data/.env                    │ New — BUREAU_PASSWORD=changeme                                              │
 ├──────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
 │ data/content.json            │ New — extracted current content                                             │
 ├──────────────────────────────┼─────────────────────────────────────────────────────────────────────────────┤
 │ assets/sw.js                 │ Minor — exclude /bureau from cache                                          │
 └──────────────────────────────┴─────────────────────────────────────────────────────────────────────────────┘

 ---
 Verification

 1. go run src/main.go must compile without errors
 2. GET / renders the page with identical content as before (no visible change)
 3. GET /bureau redirects to /bureau/login (no cookie)
 4. POST /bureau/login with wrong password shows error; correct password sets cookie and redirects to /bureau/edit
 5. /bureau/edit shows the editor with all current content pre-filled
 6. Edit intro text in Quill, save → GET / immediately reflects the change
 7. Edit a candidate name, save → table on GET / reflects change
 8. GET /bureau/logout clears cookie, redirects to /bureau/login
 9. Accessing /bureau/edit after logout redirects to login
╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌╌

