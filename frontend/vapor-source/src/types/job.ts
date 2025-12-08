export interface Job {
  jobId: string;
  title: string;
  company: string;
  location: string;
  modality?: string;
  postedDate: string;
  expiresDate?: string;
  postedTime: string;
  salary: string;
  url: string;
  minYearsExperience?: number;
  minDegree?: string;
  domain?: string;
  description?: string;
  parsedDescription?: string;
  s3Pointer?: string;
  languages?: string[];
  technologies?: string[];
  IsSoftwareEngineerRelated: boolean;
}