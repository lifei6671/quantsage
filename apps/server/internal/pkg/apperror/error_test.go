package apperror

import (
	"errors"
	"testing"
)

func TestCodeOf(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want int
	}{
		{
			name: "nil error",
			err:  nil,
			want: CodeOK,
		},
		{
			name: "app error",
			err:  New(CodeBadRequest, errors.New("bad request payload")),
			want: CodeBadRequest,
		},
		{
			name: "wrapped app error",
			err:  errors.Join(errors.New("outer"), New(CodeDatabaseError, errors.New("db down"))),
			want: CodeDatabaseError,
		},
		{
			name: "unknown error",
			err:  errors.New("boom"),
			want: CodeInternal,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := CodeOf(tc.err)
			if got != tc.want {
				t.Fatalf("CodeOf() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestMessageOf(t *testing.T) {
	t.Parallel()

	errmsg, toast := MessageOf(CodeValidationFailed)
	if errmsg != "validation failed" {
		t.Fatalf("errmsg = %q, want %q", errmsg, "validation failed")
	}
	if toast != "提交内容不符合要求" {
		t.Fatalf("toast = %q, want %q", toast, "提交内容不符合要求")
	}

	errmsg, toast = MessageOf(123456789)
	if errmsg != "internal error" {
		t.Fatalf("fallback errmsg = %q, want %q", errmsg, "internal error")
	}
	if toast != "系统异常，请稍后重试" {
		t.Fatalf("fallback toast = %q, want %q", toast, "系统异常，请稍后重试")
	}
}
