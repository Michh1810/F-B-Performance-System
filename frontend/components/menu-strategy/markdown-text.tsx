import ReactMarkdown from "react-markdown"
import remarkGfm from "remark-gfm"

import { cn } from "@/lib/utils"

// Trend/Financial Agent narratives come back as LLM-authored markdown
// (headers, bold, lists) — rendered here rather than dumped as raw text
// with literal "**"/"###" characters.
export function MarkdownText({ children, className }: { children: string; className?: string }) {
  return (
    <div
      className={cn(
        "text-sm leading-relaxed text-muted-foreground",
        "[&_strong]:font-semibold [&_strong]:text-foreground",
        "[&_h1]:mt-3 [&_h1]:mb-1 [&_h1]:font-heading [&_h1]:text-sm [&_h1]:font-semibold [&_h1]:text-foreground",
        "[&_h2]:mt-3 [&_h2]:mb-1 [&_h2]:font-heading [&_h2]:text-sm [&_h2]:font-semibold [&_h2]:text-foreground",
        "[&_h3]:mt-2 [&_h3]:mb-1 [&_h3]:font-semibold [&_h3]:text-foreground",
        "[&_p]:mt-2 first:[&_p]:mt-0",
        "[&_ul]:mt-2 [&_ul]:list-disc [&_ul]:pl-5",
        "[&_ol]:mt-2 [&_ol]:list-decimal [&_ol]:pl-5",
        "[&_li]:mt-1",
        "[&_hr]:my-3 [&_hr]:border-border",
        "[&_code]:rounded [&_code]:bg-muted [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-xs",
        "[&_a]:text-primary [&_a]:underline [&_a]:underline-offset-2",
        className
      )}
    >
      <ReactMarkdown remarkPlugins={[remarkGfm]}>{children}</ReactMarkdown>
    </div>
  )
}
