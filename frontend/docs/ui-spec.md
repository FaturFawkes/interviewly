# AI Interview Coach — Figma-Ready UI Specification

## 1) Design Foundation

### Aesthetic Direction
- Futuristic AI startup UI
- Dark premium SaaS interface
- Subtle glassmorphism with gradient glow accents
- Large whitespace and rounded cards

### Core Tokens
- Background: `#0B0F14`
- Gradient: `#7B61FF → #2F80ED`
- Secondary accent: `#00E5FF`
- Text primary: `#FFFFFF`
- Text secondary: `#A5B0C2`
- Radius: 16px / 20px / 24px
- Spacing scale: 8px base system

### Typography
- Font family: Geist / Inter / Satoshi-compatible
- Heading weights: 600–700
- Body weights: 400–500

### Grid & Layout
- 12-column grid for desktop content frames
- Content shell max width: 1280px
- Section spacing: 64px–96px on desktop, 40px–56px on mobile

## 2) Frame List (Figma)

Create five top-level desktop frames + variants for tablet/mobile:

1. `01_Landing`
2. `02_Dashboard`
3. `03_Upload_JD`
4. `04_Interview_Practice`
5. `05_Progress_Analytics`

Recommended breakpoint frames:
- Desktop: 1440×1024
- Tablet: 1024×1366
- Mobile: 390×844

## 3) Screen Blueprints

## `01_Landing`

Sections (top-to-bottom):
1. Navbar: logo left, login button right
2. Hero: headline + value proposition + CTA + preview panel
3. Feature grid: 3 cards (AI mock interview, JD analysis, feedback scoring)
4. How it works: 3-step cards
5. Testimonials: 3 testimonial cards
6. Pricing: 2 pricing cards
7. Footer

Hero CTA copy:
- Primary: “Start practicing interviews”

## `02_Dashboard`

Layout:
- Left sidebar navigation
- Top header bar
- Main content in card grid

Content blocks:
- Interview readiness card
- Average score card
- Sessions completed card
- Recent practice sessions card
- Score history chart card
- Strength vs weakness chart card
- Recommended actions card

## `03_Upload_JD`

Layout blocks:
- Resume upload card (text/file input)
- Job description input card
- Analyze button
- Parsed keyword panel
- Extracted skill tags

## `04_Interview_Practice`

Layout blocks:
- Setup card (resume + JD)
- AI interviewer question card
- Timer + score badge
- Answer input area (text + voice affordance)
- Submit answer button
- AI feedback panel (score, strengths, weaknesses, improvements, STAR feedback)
- Next question button

## `05_Progress_Analytics`

Layout blocks:
- Interview readiness score card
- Score history chart
- Strength vs weakness chart
- Recommended improvement areas card
- Weak-area tags

## 4) Reusable Components in Figma Library

Create component sets:

- `Button / Primary`
- `Button / Secondary`
- `Button / Ghost`
- `Card / Glass`
- `Input / Text`
- `Input / Textarea`
- `Sidebar / Item`
- `Navbar / Default`
- `Chart / Card`
- `Tag / Skill`
- `ScoreBadge / Variants`
- `ProgressBar / Gradient`

State variants:
- Default
- Hover
- Active
- Disabled

## 5) Interaction & Motion Guidelines

- Use subtle 180–300ms ease transitions
- Keep hover motion under 4px translate
- Use gentle glow increase on primary CTA hover
- Background orbs can drift with slow looping motion

## 6) Handoff Checklist

- [ ] All five screens at desktop/tablet/mobile
- [ ] Auto layout enabled for cards and section stacks
- [ ] Components use tokenized colors/radius/spacing
- [ ] Buttons, tags, score badges use variants
- [ ] Chart cards use consistent title/subtitle pattern
- [ ] Header/sidebar spacing matches 8px spacing system
- [ ] Export-ready for frontend parity with current implementation