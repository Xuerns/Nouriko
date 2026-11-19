export type Language = "id" | "en";
export type Theme = "light" | "dark";
export type Gender = "male" | "female";
export type ActivityLevel = "light" | "moderate" | "heavy";

export interface ProfileDTO {
  id: string;
  email: string;
  profile_url: string;
  height_cm: number | null;
  weight_kg: number | null;
  age: number | null;
  gender: Gender | null;
  activity_level: ActivityLevel | null;
  daily_calorie_target: number | null;
  target_protein_g: number | null;
  target_carbs_g: number | null;
  target_fat_g: number | null;
  target_fiber_g: number | null;
  language: Language;
  theme: Theme;
  onboarding_completed: boolean;
}

export interface RegisterResponse {
  status: string;
  user_id: string;
  email: string;
  dev_verification_token?: string;
}

export interface LoginResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
  profile: ProfileDTO;
}

export interface StatusResponse {
  status: string;
  dev_reset_token?: string;
}

export interface ProfileResponse {
  profile: ProfileDTO;
}

export interface OnboardingPayload {
  height_cm: number;
  weight_kg: number;
  age: number;
  gender: Gender;
  activity_level: ActivityLevel;
}

export interface UpdateProfilePayload {
  height_cm?: number;
  weight_kg?: number;
  age?: number;
  gender?: Gender;
  activity_level?: ActivityLevel;
  daily_calorie_target?: number;
  target_protein_g?: number;
  target_carbs_g?: number;
  target_fat_g?: number;
  target_fiber_g?: number;
  language?: Language;
  theme?: Theme;
}
