"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ArrowRight, Bot, CheckCircle, ChevronRight, MessageSquare, Send, User } from "lucide-react";

import { AppShell } from "@/components/layout/AppShell";
import { FeedbackPanel } from "@/components/interview/FeedbackPanel";
import { InterviewPanel } from "@/components/interview/InterviewPanel";
import { Button } from "@/components/ui/Button";
import { GlassCard, GradientBorderCard } from "@/components/ui/GlassCard";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { Input, TextArea } from "@/components/ui/Input";
import { useInterviewFlow } from "@/hooks/useInterviewFlow";

export default function PracticePage() {
	const router = useRouter();
	const {
		questions,
		currentQuestion,
		currentIndex,
		answer,
		setAnswer,
		feedback,
		lastScore,
		loading,
		error,
		timerSeconds,
		initializeInterview,
		submitCurrentAnswer,
		completeSession,
		goToNextQuestion,
		sessionCompleted,
	} = useInterviewFlow();

	const isLastQuestion = currentIndex >= questions.length - 1;

	return (
		<AppShell title="Interview Practice" subtitle="Text mode for structured interview training.">
			<div className="space-y-4">
				<GlassCard className="space-y-3 p-5">
					<h2 className="text-base font-semibold text-white">Choose mode</h2>
					<p className="text-sm text-[var(--color-text-muted)]">Pick how you want to practice your interview session.</p>
					<div className="flex flex-wrap gap-2">
						<Button disabled>Text mode</Button>
						<Button variant="secondary" onClick={() => router.push("/practice/voice")}>Voice mode</Button>
					</div>
				</GlassCard>

				<GlassCard className="space-y-4 p-6">
					<h2 className="text-base font-semibold text-white">Interview setup</h2>
					<p className="text-sm text-[var(--color-text-muted)]">
						Start by providing your target job description.
					</p>
					<SetupForm onStart={initializeInterview} loading={loading} interviewMode="text" />
					{error && <p className="text-sm text-red-300">{error}</p>}
				</GlassCard>

				{currentQuestion && (
					<>
						<div className="grid gap-4 lg:grid-cols-3">
							<GradientBorderCard className="lg:col-span-2">
								<div className="p-5">
									<div className="mb-4 flex items-center gap-2">
										<Bot className="h-4 w-4 text-purple-300" />
										<p className="text-sm text-purple-300">AI Interviewer</p>
									</div>
									<p className="text-white/90 text-sm">{currentQuestion.question}</p>
								</div>
							</GradientBorderCard>

							<GlassCard className="p-5">
								<p className="text-white/40 text-xs uppercase tracking-wide">Question</p>
								<p className="text-2xl text-white mt-1">{currentIndex + 1}<span className="text-base text-white/40">/{questions.length}</span></p>
								<div className="mt-4">
									<ProgressBar value={((currentIndex + 1) / questions.length) * 100} />
								</div>
								<p className="mt-3 text-xs text-white/40">{timerSeconds}s elapsed</p>
							</GlassCard>
						</div>

						<InterviewPanel
							question={currentQuestion.question}
							type={currentQuestion.type}
							timerSeconds={timerSeconds}
							current={currentIndex + 1}
							total={questions.length}
							currentScore={lastScore}
						/>

						<GlassCard className="space-y-4 p-6">
							<div className="flex items-center gap-2">
								<User className="h-4 w-4 text-cyan-300" />
								<h3 className="text-base font-semibold text-white">Your answer</h3>
							</div>

							<TextArea
								value={answer}
								onChange={(event) => setAnswer(event.target.value)}
								placeholder="Type your interview answer here..."
								className="min-h-40"
							/>

							<div className="flex flex-wrap gap-2">
								<Button onClick={() => void submitCurrentAnswer()} disabled={loading || !answer.trim()}>
									<Send className="mr-2 h-4 w-4" />
									{loading ? "Submitting..." : "Submit answer"}
								</Button>
								<Button
									variant="secondary"
									onClick={goToNextQuestion}
									disabled={currentIndex >= questions.length - 1}
								>
									<ChevronRight className="mr-2 h-4 w-4" />
									Next question
								</Button>
								{isLastQuestion && !sessionCompleted && (
									<Button variant="secondary" onClick={() => void completeSession()} disabled={loading}>
										<CheckCircle className="mr-2 h-4 w-4" />
										Finish session
									</Button>
								)}
								{sessionCompleted && (
									<div className="inline-flex items-center gap-2 rounded-full border border-green-400/30 bg-green-400/10 px-3 py-1 text-xs text-green-300">
										<CheckCircle className="h-3.5 w-3.5" />
										Session completed
									</div>
								)}
							</div>

							{!feedback && (
								<div className="flex flex-col items-center justify-center h-full min-h-[120px] rounded-xl border border-white/[0.06] bg-white/[0.02]">
									<div className="w-12 h-12 rounded-2xl bg-white/[0.03] border border-white/[0.06] flex items-center justify-center mb-3">
										<MessageSquare className="w-5 h-5 text-white/20" />
									</div>
									<p className="text-white/35 text-sm text-center">
										Submit your answer to receive<br />
										AI-powered feedback and scoring
									</p>
								</div>
							)}
						</GlassCard>

						{feedback && (
							<FeedbackPanel
								score={feedback.score}
								strengths={feedback.strengths}
								weaknesses={feedback.weaknesses}
								improvements={feedback.improvements}
								starFeedback={feedback.star_feedback}
							/>
						)}
					</>
				)}
			</div>
		</AppShell>
	);
}

function SetupForm({
	onStart,
	loading,
	interviewMode,
}: {
	onStart: (payload: { jobDescription: string; interviewMode: "text" | "voice"; targetRole: string; targetCompany: string }) => Promise<void>;
	loading: boolean;
	interviewMode: "text" | "voice";
}) {
	const [jobDescription, setJobDescription] = useState("");
	const [targetRole, setTargetRole] = useState("");
	const [targetCompany, setTargetCompany] = useState("");

	return (
		<div className="space-y-3">
			<div className="grid gap-3 md:grid-cols-2">
				<Input
					value={targetRole}
					onChange={(event) => setTargetRole(event.target.value)}
					placeholder="Target role (e.g. Senior Frontend Engineer)"
				/>
				<Input
					value={targetCompany}
					onChange={(event) => setTargetCompany(event.target.value)}
					placeholder="Target company (optional)"
				/>
			</div>
			<TextArea
				value={jobDescription}
				onChange={(event) => setJobDescription(event.target.value)}
				placeholder="Paste job description..."
				className="min-h-28"
			/>
			<Button
				onClick={() =>
					void onStart({
						jobDescription,
						interviewMode,
						targetRole,
						targetCompany,
					})
				}
				disabled={loading || !jobDescription.trim()}
			>
				{loading ? "Preparing interview..." : "Start AI interview"}
				{!loading && <ArrowRight className="ml-2 h-4 w-4" />}
			</Button>
			<p className="text-xs text-white/35 flex items-center gap-1">
				<MessageSquare className="h-3.5 w-3.5" />
				CV is pulled from latest saved profile on backend.
			</p>
		</div>
	);
}
