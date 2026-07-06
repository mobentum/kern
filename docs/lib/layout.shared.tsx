import type { BaseLayoutProps } from "fumadocs-ui/layouts/shared";

export const gitConfig = {
    user: "mobentum",
    repo: "kern",
    branch: "main",
};

export function baseOptions(): BaseLayoutProps {
    return {
        nav: {
            title: (
                <span className="text-2xl font-extrabold tracking-tight">
                    Kern
                </span>
            ),
        },
        githubUrl: `https://github.com/${gitConfig.user}/${gitConfig.repo}`,
    };
}
