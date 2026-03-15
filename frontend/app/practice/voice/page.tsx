"use client";

import { ArrowRight, Mic, Phone } from "lucide-react";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

import { AppShell } from "@/components/layout/AppShell";
import { FeedbackPanel } from "@/components/interview/FeedbackPanel";
import { InterviewPanel } from "@/components/interview/InterviewPanel";
import { Button } from "@/components/ui/Button";
import { GlassCard, GradientBorderCard } from "@/components/ui/GlassCard";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { Input, TextArea } from "@/components/ui/Input";
import { useInterviewFlow } from "@/hooks/useInterviewFlow";

export default function VoicePracticePage() {
	const router = useRouter();
	const {
		session,
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
	} = useInterviewFlow({ storageKey: "interview-flow-voice" });
	const isLastQuestion = currentIndex >= questions.length - 1;
	const interviewLanguageLabel = session?.interview_language === "id" ? "Bahasa Indonesia" : "English";

	useEffect(() => {
		let cancelled = false;

		if (typeof window === "undefined") {
			return;
		}

		const pendingSetupRaw = window.sessionStorage.getItem("pending-practice-voice-setup");
		if (!pendingSetupRaw) {
			return;
		}

		window.sessionStorage.removeItem("pending-practice-voice-setup");

		try {
			const pendingSetup = JSON.parse(pendingSetupRaw) as {
				jobDescription?: string;
				interviewLanguage?: "id" | "en";
				interviewDifficulty?: "easy" | "medium" | "hard";
				targetRole?: string;
				targetCompany?: string;
			};
			const pendingJobDescription = pendingSetup.jobDescription?.trim();

			if (!pendingJobDescription) {
				return;
			}

			void (async () => {
				const initialized = await initializeInterview({
					jobDescription: pendingJobDescription,
					interviewMode: "voice",
					interviewLanguage: pendingSetup.interviewLanguage,
					interviewDifficulty: pendingSetup.interviewDifficulty,
					targetRole: pendingSetup.targetRole,
					targetCompany: pendingSetup.targetCompany,
				});

				if (!cancelled && initialized) {
					router.replace("/practice/voice/call");
				}
			})();
		} catch {
			return;
		}

		return () => {
			cancelled = true;
		};
	}, [initializeInterview, router]);

	function openCallScreen() {
		router.push("/practice/voice/call");
	}

	return (
		<AppShell title="Interview Practice" subtitle="Voice mode with backend ElevenLabs call interaction.">
			<div className="space-y-4">
				<GlassCard className="space-y-3 p-5">
					<h2 className="text-base font-semibold text-white">Choose mode</h2>
					<p className="text-sm text-[var(--color-text-muted)]">Switch between text-based or call-style interview practice.</p>
					<div className="flex flex-wrap gap-2">
						<Button variant="secondary" onClick={() => router.push("/practice")}>Text mode</Button>
						<Button disabled>Voice mode</Button>
					</div>
				</GlassCard>

				<GlassCard className="space-y-4 p-6">
					<h2 className="text-base font-semibold text-white">Interview setup</h2>
					<p className="text-sm text-[var(--color-text-muted)]">
						Start by providing your target job description.
					</p>
					<SetupForm onStart={initializeInterview} loading={loading} interviewMode="voice" />
					{error && <p className="text-sm text-red-300">{error}</p>}
				</GlassCard>

				{currentQuestion && (
					<>
						<GradientBorderCard>
							<div className="p-5 space-y-3">
								<div className="flex items-center justify-between gap-3">
									<div>
										<h3 className="text-base font-semibold text-white">Voice call mode</h3>
										<p className="text-sm text-[var(--color-text-muted)]">Open full-screen call interaction inspired by Siri style UI.</p>
									</div>
									<div className="flex flex-col items-end gap-1">
										<span className="rounded-full border border-cyan-300/30 bg-cyan-400/10 px-3 py-1 text-xs text-cyan-200">
											{sessionCompleted ? "Session completed" : "Ready"}
										</span>
										<span className="rounded-full border border-white/15 bg-white/[0.03] px-3 py-1 text-[11px] text-white/70">
											Language: {interviewLanguageLabel}
										</span>
									</div>
								</div>

								<div className="flex flex-wrap gap-2">
									<Button onClick={openCallScreen} className="shadow-[0_0_20px_rgba(6,182,212,0.2)]">
										<Phone className="mr-2 h-4 w-4" />
										Start call
									</Button>
									<Button variant="secondary" onClick={openCallScreen}>
										<ArrowRight className="mr-2 h-4 w-4" />
										Open full screen
									</Button>
								</div>

								<p className="text-xs text-[var(--color-text-muted)]">
									Tap Start call to continue interview in immersive call page.
								</p>
							</div>
						</GradientBorderCard>

						<InterviewPanel
							question={currentQuestion.question}
							type={currentQuestion.type}
							timerSeconds={timerSeconds}
							current={currentIndex + 1}
							total={questions.length}
							currentScore={lastScore}
						/>

						<GlassCard className="space-y-4 p-6">
							<div className="flex items-center justify-between">
								<h3 className="text-base font-semibold text-white">Your answer</h3>
								<p className="text-xs text-[var(--color-text-muted)]">Type here, or continue in call page for voice interaction.</p>
							</div>

							<TextArea
								value={answer}
								onChange={(event) => setAnswer(event.target.value)}
								placeholder="Type your interview answer here..."
								className="min-h-40"
							/>

							<div className="flex flex-wrap gap-2">
								<Button variant="secondary" onClick={openCallScreen}>
									<Phone className="mr-2 h-4 w-4" />
									Go to call screen
								</Button>
								<Button onClick={() => void submitCurrentAnswer()} disabled={loading || !answer.trim()}>
									{loading ? "Submitting..." : "Submit answer"}
								</Button>
								<Button
									variant="secondary"
									onClick={goToNextQuestion}
									disabled={currentIndex >= questions.length - 1}
								>
									Next question
								</Button>
								{isLastQuestion && !sessionCompleted && (
									<Button variant="secondary" onClick={() => void completeSession()} disabled={loading}>
										Finish session
									</Button>
								)}
								{sessionCompleted && (
									<div className="inline-flex items-center gap-2 rounded-full border border-green-400/30 bg-green-400/10 px-3 py-1 text-xs text-green-300">
										Session completed
									</div>
								)}
							</div>

							<div>
								<p className="mb-2 text-xs uppercase tracking-wide text-[var(--color-text-muted)]">Interview progress</p>
								<ProgressBar value={((currentIndex + 1) / questions.length) * 100} />
							</div>

							{!feedback && (
								<div className="flex flex-col items-center justify-center h-full min-h-[120px] rounded-xl border border-white/[0.06] bg-white/[0.02]">
									<div className="w-12 h-12 rounded-2xl bg-white/[0.03] border border-white/[0.06] flex items-center justify-center mb-3">
										<Mic className="w-5 h-5 text-white/20" />
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
	onStart: (payload: {
		jobDescription: string;
		interviewMode: "text" | "voice";
		interviewLanguage: "id" | "en";
		interviewDifficulty?: "easy" | "medium" | "hard";
		targetRole: string;
		targetCompany: string;
	}) => Promise<boolean>;
	loading: boolean;
	interviewMode: "text" | "voice";
}) {
	const [jobDescription, setJobDescription] = useState("");
	const [targetRole, setTargetRole] = useState("");
	const [targetCompany, setTargetCompany] = useState("");
	const [interviewLanguage, setInterviewLanguage] = useState<"id" | "en">("id");

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
			<div className="space-y-2">
				<p className="text-xs uppercase tracking-wide text-white/45">Interview language</p>
				<div className="flex flex-wrap gap-2">
					<Button type="button" variant={interviewLanguage === "id" ? "primary" : "secondary"} onClick={() => setInterviewLanguage("id")}>
						Bahasa Indonesia
					</Button>
					<Button type="button" variant={interviewLanguage === "en" ? "primary" : "secondary"} onClick={() => setInterviewLanguage("en")}>
						English
					</Button>
				</div>
			</div>
			<Button
				onClick={() =>
					void onStart({
						jobDescription,
						interviewMode,
						interviewLanguage,
						targetRole,
						targetCompany,
					})
				}
				disabled={loading || !jobDescription.trim()}
			>
				{loading ? "Preparing interview..." : "Start AI interview"}
			</Button>
			<p className="text-xs text-white/35">
				CV is pulled from latest saved profile on backend.
			</p>
		</div>
	);
}
