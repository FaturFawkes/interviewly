"use client";

import { Mic, Phone, PhoneOff, Volume2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";

import { AppShell } from "@/components/layout/AppShell";
import { FeedbackPanel } from "@/components/interview/FeedbackPanel";
import { InterviewPanel } from "@/components/interview/InterviewPanel";
import { Button } from "@/components/ui/Button";
import { GlassCard, GradientBorderCard } from "@/components/ui/GlassCard";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { Input, TextArea } from "@/components/ui/Input";
import { getAuthToken } from "@/lib/auth/token-provider";
import { useInterviewFlow } from "@/hooks/useInterviewFlow";

export default function VoicePracticePage() {
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

	const apiBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL ?? "/api-proxy";
	const mediaRecorderRef = useRef<MediaRecorder | null>(null);
	const mediaStreamRef = useRef<MediaStream | null>(null);
	const voiceChunksRef = useRef<BlobPart[]>([]);
	const audioPlayerRef = useRef<HTMLAudioElement | null>(null);
	const lastSpokenQuestionRef = useRef<string>("");

	const [isCallActive, setIsCallActive] = useState(false);
	const [isListening, setIsListening] = useState(false);
	const [isSpeaking, setIsSpeaking] = useState(false);
	const [voiceError, setVoiceError] = useState<string | null>(null);
	const [voiceInfo, setVoiceInfo] = useState<string | null>(null);

	const recordingSupported = useMemo(() => {
		if (typeof window === "undefined") {
			return false;
		}
		return Boolean(window.MediaRecorder && navigator.mediaDevices?.getUserMedia);
	}, []);

	const stopVoiceInput = useCallback(() => {
		if (mediaRecorderRef.current && mediaRecorderRef.current.state !== "inactive") {
			mediaRecorderRef.current.stop();
		}
	}, []);

	const stopQuestionAudio = useCallback(() => {
		if (audioPlayerRef.current) {
			audioPlayerRef.current.pause();
			audioPlayerRef.current = null;
		}
		setIsSpeaking(false);
	}, []);

	useEffect(() => {
		return () => {
			stopVoiceInput();
			if (mediaStreamRef.current) {
				mediaStreamRef.current.getTracks().forEach((track) => track.stop());
			}
			stopQuestionAudio();
		};
	}, [stopQuestionAudio, stopVoiceInput]);

	async function transcribeRecordedAudio(blob: Blob) {
		const token = await getAuthToken();
		if (!token) {
			throw new Error("Authentication token is missing.");
		}

		const formData = new FormData();
		formData.append("audio", blob, "answer.webm");

		const response = await fetch(`${apiBaseUrl}/api/voice/stt`, {
			method: "POST",
			headers: {
				Authorization: `Bearer ${token}`,
			},
			body: formData,
		});

		const payload = (await response.json().catch(() => ({}))) as { text?: string; error?: string };
		if (!response.ok) {
			throw new Error(payload.error ?? "Failed to transcribe voice.");
		}

		return (payload.text ?? "").trim();
	}

	async function startVoiceInput() {
		setVoiceError(null);
		setVoiceInfo(null);

		if (!recordingSupported || typeof window === "undefined") {
			setVoiceError("Voice input is not supported in this browser.");
			return;
		}

		try {
			const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
			mediaStreamRef.current = stream;
			voiceChunksRef.current = [];

			const recorder = new MediaRecorder(stream);
			mediaRecorderRef.current = recorder;

			recorder.ondataavailable = (event) => {
				if (event.data && event.data.size > 0) {
					voiceChunksRef.current.push(event.data);
				}
			};

			recorder.onstop = () => {
				void (async () => {
					try {
						const audioBlob = new Blob(voiceChunksRef.current, { type: "audio/webm" });
						const transcript = await transcribeRecordedAudio(audioBlob);
						if (!transcript) {
							setVoiceError("No speech detected. Please try again.");
							return;
						}
						setAnswer((prev) => {
							const trimmed = prev.trim();
							return trimmed ? `${trimmed} ${transcript}` : transcript;
						});
						setVoiceInfo("Voice converted to text successfully.");
					} catch (voiceProcessError) {
						setVoiceError(voiceProcessError instanceof Error ? voiceProcessError.message : "Failed to process voice input.");
					} finally {
						if (mediaStreamRef.current) {
							mediaStreamRef.current.getTracks().forEach((track) => track.stop());
							mediaStreamRef.current = null;
						}
						setIsListening(false);
					}
				})();
			};

			recorder.start();
			setIsListening(true);
		} catch {
			setVoiceError("Microphone permission denied or unavailable.");
		}
	}

	const speakCurrentQuestion = useCallback(async () => {
		setVoiceError(null);
		setVoiceInfo(null);

		if (!currentQuestion?.question) {
			return;
		}

		try {
			const token = await getAuthToken();
			if (!token) {
				throw new Error("Authentication token is missing.");
			}

			setIsSpeaking(true);

			const response = await fetch(`${apiBaseUrl}/api/voice/tts`, {
				method: "POST",
				headers: {
					"Content-Type": "application/json",
					Authorization: `Bearer ${token}`,
				},
				body: JSON.stringify({ text: currentQuestion.question }),
			});

			if (!response.ok) {
				const payload = (await response.json().catch(() => ({}))) as { error?: string };
				throw new Error(payload.error ?? "Failed to generate voice output.");
			}

			const audioBlob = await response.blob();
			const audioUrl = URL.createObjectURL(audioBlob);

			stopQuestionAudio();

			const player = new Audio(audioUrl);
			audioPlayerRef.current = player;
			player.onended = () => {
				setIsSpeaking(false);
				URL.revokeObjectURL(audioUrl);
			};
			player.onerror = () => {
				setIsSpeaking(false);
				setVoiceError("Unable to play generated voice.");
				URL.revokeObjectURL(audioUrl);
			};

			await player.play();
		} catch (ttsError) {
			setIsSpeaking(false);
			setVoiceError(ttsError instanceof Error ? ttsError.message : "Unable to read the question aloud.");
		}
	}, [apiBaseUrl, currentQuestion?.question, stopQuestionAudio]);

	useEffect(() => {
		if (!isCallActive || !currentQuestion?.question) {
			return;
		}

		if (lastSpokenQuestionRef.current === currentQuestion.question) {
			return;
		}

		lastSpokenQuestionRef.current = currentQuestion.question;
		void speakCurrentQuestion();
	}, [isCallActive, currentQuestion?.question, speakCurrentQuestion]);

	function startCall() {
		setVoiceError(null);
		setVoiceInfo("Call connected. The interviewer will read each question automatically.");
		setIsCallActive(true);
		if (currentQuestion?.question) {
			lastSpokenQuestionRef.current = "";
		}
	}

	function endCall() {
		stopVoiceInput();
		stopQuestionAudio();
		setIsCallActive(false);
		setVoiceInfo("Call ended.");
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
							<div className="flex items-center justify-between">
								<div>
									<h3 className="text-base font-semibold text-white">Voice call mode</h3>
									<p className="text-sm text-[var(--color-text-muted)]">Call-style interaction (similar to ChatGPT call experience).</p>
								</div>
								<span className="rounded-full border border-cyan-300/30 bg-cyan-400/10 px-3 py-1 text-xs text-cyan-200">
									{isCallActive ? (isSpeaking ? "Agent speaking" : isListening ? "You speaking" : "Connected") : "Disconnected"}
								</span>
							</div>

							<div className="flex flex-wrap gap-2">
								{isCallActive ? (
									<Button variant="secondary" onClick={endCall} className="border-red-400/40 text-red-200 hover:border-red-300/70">
										<PhoneOff className="mr-2 h-4 w-4" />
										End call
									</Button>
								) : (
									<Button onClick={startCall} className="shadow-[0_0_20px_rgba(6,182,212,0.2)]">
										<Phone className="mr-2 h-4 w-4" />
										Start call
									</Button>
								)}

								<Button
									variant="secondary"
									onClick={isListening ? stopVoiceInput : startVoiceInput}
									disabled={!isCallActive || isSpeaking}
								>
									<Mic className="mr-2 h-4 w-4" />
									{isListening ? "Stop talking" : "Talk"}
								</Button>

								<Button
									variant="secondary"
									onClick={speakCurrentQuestion}
									disabled={!isCallActive || isSpeaking || isListening}
								>
									<Volume2 className="mr-2 h-4 w-4" />
									{isSpeaking ? "Reading..." : "Replay question"}
								</Button>
							</div>

							{!isCallActive && (
								<p className="text-xs text-[var(--color-text-muted)]">
									Start call to enable voice interaction.
								</p>
							)}
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
								<p className="text-xs text-[var(--color-text-muted)]">Type or use call controls above to speak.</p>
							</div>

							{voiceError && <p className="text-sm text-red-300">{voiceError}</p>}
							{voiceInfo && <p className="text-sm text-cyan-200">{voiceInfo}</p>}
							{isListening && <p className="text-xs text-cyan-200">Recording... click Stop talking to transcribe your response.</p>}

							<TextArea
								value={answer}
								onChange={(event) => setAnswer(event.target.value)}
								placeholder="Type your interview answer here..."
								className="min-h-40"
							/>

							<div className="flex flex-wrap gap-2">
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
			</Button>
			<p className="text-xs text-white/35">
				CV is pulled from latest saved profile on backend.
			</p>
		</div>
	);
}
