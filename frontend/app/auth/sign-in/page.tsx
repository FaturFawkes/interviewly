import { SignInView } from "./SignInView";

type SignInPageProps = {
  searchParams?: Promise<Record<string, string | string[] | undefined>>;
};

export default async function SignInPage({ searchParams }: SignInPageProps) {
  const resolved = (await searchParams) ?? {};
  const callbackValue = resolved.callbackUrl;
  const callbackUrl = Array.isArray(callbackValue)
    ? callbackValue[0] ?? "/dashboard"
    : callbackValue ?? "/dashboard";

  return <SignInView callbackUrl={callbackUrl} />;
}
