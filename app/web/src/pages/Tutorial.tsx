import { useEffect, useRef, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { ImageOff, BookOpen, ChevronDown, ChevronRight } from 'lucide-react'

import { ScrollArea } from '@/components/ui/scroll-area'
import { cn } from '@/lib/utils'
import { groups, sections, type TutorialGroup } from '@/tutorial/sections'

/**
 * Tutorial — single-page operator manual with sticky TOC + scroll-spy.
 *
 * Sections live as markdown files under src/tutorial/sections/ and
 * are loaded automatically via Vite's import.meta.glob (see
 * tutorial/sections.ts). Add a new file to add a new section — no
 * registration needed.
 *
 * Image strategy: markdown `![alt](/tutorial/foo.png)` references
 * static assets under public/tutorial/. When the asset is missing,
 * `TutorialImage` shows a placeholder card displaying the alt text
 * so authoring a section before the screenshots exist still reads
 * sensibly.
 */
export function TutorialPage() {
	const [activeId, setActiveId] = useState<string>(sections[0]?.id ?? '')
	const observerRef = useRef<IntersectionObserver | null>(null)
	const sectionRefs = useRef<Record<string, HTMLElement | null>>({})

	// Wire up scroll-spy: highlight the TOC entry whose section is
	// currently most-visible in the viewport.
	useEffect(() => {
		if (sections.length === 0) return
		// rootMargin shifts the "visible" region up by 25% of the
		// viewport so the next heading flips before it crosses the
		// physical centre — feels more like real reading.
		const observer = new IntersectionObserver(
			(entries) => {
				const visible = entries
					.filter((e) => e.isIntersecting)
					.sort((a, b) => b.intersectionRatio - a.intersectionRatio)
				if (visible.length > 0) {
					const id = visible[0].target.id
					if (id) setActiveId(id)
				}
			},
			{ rootMargin: '-25% 0px -55% 0px', threshold: [0, 0.25, 0.5, 0.75, 1] },
		)
		observerRef.current = observer
		for (const s of sections) {
			const el = sectionRefs.current[s.id]
			if (el) observer.observe(el)
		}
		return () => observer.disconnect()
	}, [])

	const handleTocClick = (id: string) => {
		const el = sectionRefs.current[id]
		if (!el) return
		setActiveId(id)
		// scroll-margin-top in the section style accounts for the
		// header bar so the heading isn't tucked under it.
		el.scrollIntoView({ behavior: 'smooth', block: 'start' })
	}

	if (sections.length === 0) {
		return (
			<div className="h-full flex items-center justify-center text-[12px] text-muted-foreground">
				No tutorial content registered.
			</div>
		)
	}

	return (
		<div className="h-full flex bg-background">
			<TutorialToc
				groups={groups}
				activeId={activeId}
				onSelect={handleTocClick}
			/>
			<ScrollArea className="flex-1">
				<article className="mx-auto max-w-[840px] px-8 py-10 prose-tutorial">
					<header className="mb-10 pb-6 border-b border-border">
						<div className="flex items-center gap-3">
							<BookOpen className="size-6 text-muted-foreground" />
							<h1 className="text-[22px] font-semibold tracking-tight">
								OpenDray operator guide
							</h1>
						</div>
						<p className="text-[13px] text-muted-foreground mt-2 leading-relaxed">
							Walks through every page of the admin: what it's for,
							how to configure it, and the gotchas that bit us first
							time round. Use the table of contents on the left to
							jump around.
						</p>
					</header>

					{sections.map((s) => (
						<section
							key={s.id}
							id={s.id}
							ref={(el) => {
								sectionRefs.current[s.id] = el
							}}
							className="scroll-mt-12 mb-16"
						>
							<ReactMarkdown
								remarkPlugins={[remarkGfm]}
								components={{
									img: TutorialImage,
									h1: H1,
									h2: H2,
									h3: H3,
									p: P,
									ul: UL,
									ol: OL,
									li: LI,
									code: Code,
									pre: Pre,
									blockquote: BlockQuote,
									table: Table,
									thead: THead,
									tbody: TBody,
									tr: TR,
									th: TH,
									td: TD,
									a: Anchor,
								}}
							>
								{s.body}
							</ReactMarkdown>
						</section>
					))}
				</article>
			</ScrollArea>
		</div>
	)
}

// ── TOC ───────────────────────────────────────────────────────────

function TutorialToc({
	groups,
	activeId,
	onSelect,
}: {
	groups: TutorialGroup[]
	activeId: string
	onSelect: (id: string) => void
}) {
	// Track which groups are expanded. Default = expand the group
	// containing the active section + (because the list is short
	// enough) all single-section groups stay open as standalone
	// links. Operator can collapse / expand each group manually.
	const [collapsed, setCollapsed] = useState<Set<string>>(() => new Set())
	const toggleGroup = (id: string) => {
		setCollapsed((prev) => {
			const next = new Set(prev)
			if (next.has(id)) next.delete(id)
			else next.add(id)
			return next
		})
	}
	// When the active section changes, force-expand its parent group
	// so the active link is always reachable in the TOC.
	useEffect(() => {
		const parent = groups.find((g) => g.sections.some((s) => s.id === activeId))
		if (!parent) return
		setCollapsed((prev) => {
			if (!prev.has(parent.id)) return prev
			const next = new Set(prev)
			next.delete(parent.id)
			return next
		})
	}, [activeId, groups])

	return (
		<nav className="hidden lg:flex w-64 shrink-0 border-r border-border bg-card/30 flex-col">
			<div className="px-4 py-4 border-b border-border">
				<div className="text-[10px] uppercase tracking-wider text-muted-foreground/70 font-medium">
					Tutorial
				</div>
				<div className="text-[13px] font-semibold mt-0.5">Table of contents</div>
			</div>
			<ScrollArea className="flex-1">
				<div className="py-3 px-2 space-y-2">
					{groups.map((g) => {
						const single = g.sections.length === 1
						const groupActive = g.sections.some((s) => s.id === activeId)
						const isCollapsed = collapsed.has(g.id) && !single

						// Single-section groups render as one flat
						// link — no group header / indent / collapse
						// adds informational value when there's
						// nothing to group.
						if (single) {
							const s = g.sections[0]
							return (
								<button
									key={g.id}
									type="button"
									onClick={() => onSelect(s.id)}
									className={cn(
										'w-full text-left h-8 px-2.5 rounded-md text-[12.5px] transition-colors',
										'text-muted-foreground/90 hover:text-foreground hover:bg-card',
										activeId === s.id && 'bg-card text-foreground font-medium',
									)}
								>
									{s.title}
								</button>
							)
						}

						return (
							<div key={g.id} className="space-y-0.5">
								<button
									type="button"
									onClick={() => toggleGroup(g.id)}
									className={cn(
										'w-full text-left h-7 px-2.5 rounded-md flex items-center gap-1.5',
										'text-[10.5px] uppercase tracking-wider font-semibold transition-colors',
										groupActive
											? 'text-foreground/95'
											: 'text-muted-foreground/70 hover:text-foreground/90',
									)}
								>
									{isCollapsed ? (
										<ChevronRight className="size-3 shrink-0" />
									) : (
										<ChevronDown className="size-3 shrink-0" />
									)}
									<span className="flex-1">{g.label}</span>
									<span className="text-[10px] font-normal opacity-60">
										{g.sections.length}
									</span>
								</button>
								{!isCollapsed && (
									<ul className="space-y-0.5 ml-3 pl-1.5 border-l border-border/60">
										{g.sections.map((s) => (
											<li key={s.id}>
												<button
													type="button"
													onClick={() => onSelect(s.id)}
													className={cn(
														'w-full text-left h-7 px-2.5 rounded-md text-[12px] transition-colors',
														'text-muted-foreground hover:text-foreground hover:bg-card',
														activeId === s.id &&
															'bg-card text-foreground font-medium',
													)}
												>
													{s.title}
												</button>
											</li>
										))}
									</ul>
								)}
							</div>
						)
					})}
				</div>
			</ScrollArea>
		</nav>
	)
}

// ── Markdown component overrides ──────────────────────────────────
//
// We cannot rely on global `prose` Tailwind plugin (not installed),
// so each block-level element gets a tiny custom class for sane
// typography. Keeps the bundle smaller and the styling here-visible.

function H1({ children }: { children?: React.ReactNode }) {
	// The first H1 already lives as the section anchor; suppress
	// duplicate rendering by emitting it visually as a section title.
	return (
		<h2 className="text-[20px] font-semibold tracking-tight mt-2 mb-4 pb-2 border-b border-border/60">
			{children}
		</h2>
	)
}
function H2({ children }: { children?: React.ReactNode }) {
	return <h3 className="text-[16px] font-semibold mt-8 mb-3">{children}</h3>
}
function H3({ children }: { children?: React.ReactNode }) {
	return <h4 className="text-[14px] font-semibold mt-6 mb-2 text-foreground/90">{children}</h4>
}
function P({ children }: { children?: React.ReactNode }) {
	return <p className="text-[13px] leading-relaxed mb-3 text-foreground/90">{children}</p>
}
function UL({ children }: { children?: React.ReactNode }) {
	return <ul className="list-disc pl-6 space-y-1 mb-3 text-[13px] leading-relaxed">{children}</ul>
}
function OL({ children }: { children?: React.ReactNode }) {
	return <ol className="list-decimal pl-6 space-y-1 mb-3 text-[13px] leading-relaxed">{children}</ol>
}
function LI({ children }: { children?: React.ReactNode }) {
	return <li className="marker:text-muted-foreground/60">{children}</li>
}
function Code({ children, className }: { children?: React.ReactNode; className?: string }) {
	const inline = !className
	if (inline) {
		return (
			<code className="px-1.5 py-0.5 rounded bg-muted/60 text-[12px] font-mono">
				{children}
			</code>
		)
	}
	return <code className={cn(className, 'font-mono text-[12px]')}>{children}</code>
}
function Pre({ children }: { children?: React.ReactNode }) {
	return (
		<pre className="bg-muted/40 border border-border rounded-md px-3 py-2.5 my-3 text-[12px] font-mono overflow-x-auto">
			{children}
		</pre>
	)
}
function BlockQuote({ children }: { children?: React.ReactNode }) {
	return (
		<blockquote className="border-l-2 border-accent/60 pl-3 py-1 my-3 text-[12.5px] text-muted-foreground italic bg-muted/20">
			{children}
		</blockquote>
	)
}
function Table({ children }: { children?: React.ReactNode }) {
	return (
		<div className="my-4 overflow-x-auto border border-border rounded-md">
			<table className="w-full text-[12px]">{children}</table>
		</div>
	)
}
function THead({ children }: { children?: React.ReactNode }) {
	return <thead className="bg-muted/30 border-b border-border">{children}</thead>
}
function TBody({ children }: { children?: React.ReactNode }) {
	return <tbody className="divide-y divide-border/40">{children}</tbody>
}
function TR({ children }: { children?: React.ReactNode }) {
	return <tr>{children}</tr>
}
function TH({ children }: { children?: React.ReactNode }) {
	return <th className="px-3 py-1.5 text-left font-medium text-foreground/90">{children}</th>
}
function TD({ children }: { children?: React.ReactNode }) {
	return <td className="px-3 py-1.5 align-top text-foreground/85">{children}</td>
}
function Anchor({
	children,
	href,
}: {
	children?: React.ReactNode
	href?: string
}) {
	const external = href?.startsWith('http://') || href?.startsWith('https://')
	return (
		<a
			href={href}
			target={external ? '_blank' : undefined}
			rel={external ? 'noreferrer noopener' : undefined}
			className="text-accent hover:underline underline-offset-2"
		>
			{children}
		</a>
	)
}

// TutorialImage renders <img>, but if the asset is missing we want a
// readable placeholder so half-finished sections still convey the
// intended layout. We can't synchronously check existence — the
// browser fetches lazily. So we attach onError and swap to a card
// that surfaces the alt text + filename.
function TutorialImage({
	src,
	alt,
}: {
	src?: string
	alt?: string
}) {
	const [errored, setErrored] = useState(false)

	if (!src) return null
	if (errored) {
		return (
			<TutorialImagePlaceholder src={src} alt={alt ?? ''} />
		)
	}
	return (
		<figure className="my-4 border border-border rounded-md overflow-hidden bg-card/30">
			<img
				src={src}
				alt={alt ?? ''}
				className="block w-full h-auto"
				onError={() => setErrored(true)}
			/>
			{alt && (
				<figcaption className="text-[11px] text-muted-foreground/80 px-3 py-1.5 border-t border-border/60">
					{alt}
				</figcaption>
			)}
		</figure>
	)
}

function TutorialImagePlaceholder({ src, alt }: { src: string; alt: string }) {
	return (
		<div className="my-4 border border-dashed border-border/80 rounded-md bg-muted/20 px-4 py-6 flex items-start gap-3">
			<ImageOff className="size-4 mt-0.5 text-muted-foreground/60 shrink-0" />
			<div className="flex-1 min-w-0">
				<div className="text-[12px] font-medium">{alt || 'Screenshot pending'}</div>
				<div className="text-[11px] text-muted-foreground/70 font-mono mt-0.5 truncate">
					{src}
				</div>
				<div className="text-[11px] text-muted-foreground/60 mt-2 leading-relaxed">
					Drop the actual screenshot at <code className="px-1 py-0.5 bg-muted/40 rounded">app/web/public{src}</code> and refresh — the placeholder swaps for the image.
				</div>
			</div>
		</div>
	)
}
