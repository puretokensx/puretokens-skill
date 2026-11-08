import { repositoryRoot, validateRepository } from "./skill-registry.mjs";
import { validatePublicAcceptance } from "./validate-publication-privacy.mjs";

const errors = await validateRepository();
errors.push(...await validatePublicAcceptance(repositoryRoot));
if (errors.length) {
  for (const error of errors) process.stderr.write(`error: ${error}\n`);
  process.exitCode = 1;
} else {
  process.stdout.write("Skill repository validation passed.\n");
}
