# FBPerformance Frontend

This is the frontend application for the FBPerformance system, built with **Next.js**, **Tailwind CSS v4**, and **shadcn/ui**.

## 🎨 Design System & Theme

The project has been initialized with a custom design system:
- **Theme:** Neutral (Light/Dark mode supported)
- **Chart Colors:** Emerald
- **Typography:** Instrument Sans (Headings) & Inter (Body text)
- **Border Radius:** Small (0.45rem)

These design tokens are centrally managed in `app/globals.css`. By changing the values in that file, you can instantly re-theme the entire application.

## 🚀 Getting Started

First, run the development server:

```bash
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

## 🧩 Adding & Using Components

This project uses **shadcn/ui**. Unlike traditional component libraries, shadcn generates the actual component code directly into your `components/ui` folder so you have full control over the design and behavior.

### 1. Add a Component
To add a new component (e.g., a Card or a Chart), run the shadcn CLI:

```bash
npx shadcn@latest add card
npx shadcn@latest add chart
```

### 2. Use the Component
Import the newly added component into your page:

```tsx
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

export default function MyPage() {
  return (
    <Card className="bg-card text-card-foreground">
      <CardHeader>
        <CardTitle className="font-heading">My Title</CardTitle>
      </CardHeader>
      <CardContent>
        This card uses the Neutral theme colors automatically!
      </CardContent>
    </Card>
  );
}
```

### 3. Using Design Tokens in Tailwind
You can use the semantic theme tokens directly in your Tailwind classes:
- **Colors:** `bg-primary`, `text-destructive`, `bg-chart-1`
- **Fonts:** `font-heading` (for Instrument Sans), `font-sans` (for Inter, which is the default)
- **Borders:** `border-border`, `rounded-md`
