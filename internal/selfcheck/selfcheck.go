// Package selfcheck 提供无需外部依赖、执行后自行退出的 --smoke-test。
// 它在进程内启动 httptest 服务，逐项验证 Cron 服务的解析、下次执行时间、
// 合法性校验与错误处理，覆盖该主题特有的边界约束。
package selfcheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"task011-cron/internal/httpapi"
)

func Run() int {
	passed, failed := 0, 0
	check := func(name string, fn func() error) {
		if err := fn(); err != nil {
			failed++
			fmt.Printf("FAIL %-34s %v\n", name, err)
		} else {
			passed++
			fmt.Printf("PASS %s\n", name)
		}
	}

	api := httpapi.New()
	srv := httptest.NewServer(api.Handler())
	defer srv.Close()

	do := func(method, path, body string) (*http.Response, []byte, error) {
		var r io.Reader
		if body != "" {
			r = bytes.NewReader([]byte(body))
		}
		req, err := http.NewRequest(method, srv.URL+path, r)
		if err != nil {
			return nil, nil, err
		}
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, nil, err
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp, data, nil
	}

	mkExpr := func(expr string) string {
		b, _ := json.Marshal(map[string]string{"expr": expr})
		return string(b)
	}
	mkNext := func(expr, from string) string {
		b, _ := json.Marshal(map[string]string{"expr": expr, "from": from})
		return string(b)
	}

	parseResp := func(data []byte) (map[string]any, error) {
		var out map[string]any
		if err := json.Unmarshal(data, &out); err != nil {
			return nil, err
		}
		return out, nil
	}

	check("健康检查", func() error {
		resp, _, err := do(http.MethodGet, "/healthz", "")
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status=%d", resp.StatusCode)
		}
		return nil
	})

	check("解析 */15 分钟", func() error {
		resp, body, err := do(http.MethodPost, "/api/cron/parse", mkExpr("*/15 * * * *"))
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status=%d body=%s", resp.StatusCode, body)
		}
		out, err := parseResp(body)
		if err != nil {
			return err
		}
		mins, _ := out["minute"].([]any)
		if len(mins) != 4 {
			return fmt.Errorf("minute=%v", mins)
		}
		got := make([]int, 0, len(mins))
		for _, m := range mins {
			f, _ := m.(float64)
			got = append(got, int(f))
		}
		want := []int{0, 15, 30, 45}
		if len(got) != len(want) {
			return fmt.Errorf("minute=%v want=%v", got, want)
		}
		for i := range want {
			if got[i] != want[i] {
				return fmt.Errorf("minute=%v want=%v", got, want)
			}
		}
		return nil
	})

	check("解析星期7等价0", func() error {
		resp, body, err := do(http.MethodPost, "/api/cron/parse", mkExpr("0 0 * * 7"))
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status=%d body=%s", resp.StatusCode, body)
		}
		out, err := parseResp(body)
		if err != nil {
			return err
		}
		dow, _ := out["day_of_week"].([]any)
		if len(dow) != 1 || int(dow[0].(float64)) != 0 {
			return fmt.Errorf("dow=%v", dow)
		}
		return nil
	})

	check("下次执行每15分钟", func() error {
		resp, body, err := do(http.MethodPost, "/api/cron/next", mkNext("*/15 * * * *", "2026-08-14T12:00:00Z"))
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status=%d body=%s", resp.StatusCode, body)
		}
		out, err := parseResp(body)
		if err != nil {
			return err
		}
		if out["next"] != "2026-08-14T12:15:00Z" {
			return fmt.Errorf("next=%v", out["next"])
		}
		return nil
	})

	check("严格晚于命中点", func() error {
		resp, body, err := do(http.MethodPost, "/api/cron/next", mkNext("*/15 * * * *", "2026-08-14T12:15:00Z"))
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status=%d body=%s", resp.StatusCode, body)
		}
		out, err := parseResp(body)
		if err != nil {
			return err
		}
		if out["next"] != "2026-08-14T12:30:00Z" {
			return fmt.Errorf("next=%v", out["next"])
		}
		return nil
	})

	check("每月1号", func() error {
		resp, body, err := do(http.MethodPost, "/api/cron/next", mkNext("0 0 1 * *", "2026-08-14T12:00:00Z"))
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status=%d body=%s", resp.StatusCode, body)
		}
		out, err := parseResp(body)
		if err != nil {
			return err
		}
		if out["next"] != "2026-09-01T00:00:00Z" {
			return fmt.Errorf("next=%v", out["next"])
		}
		return nil
	})

	check("1号或周一(析取)", func() error {
		resp, body, err := do(http.MethodPost, "/api/cron/next", mkNext("0 0 1 * 1", "2026-08-14T12:00:00Z"))
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status=%d body=%s", resp.StatusCode, body)
		}
		out, err := parseResp(body)
		if err != nil {
			return err
		}
		// 2026-08-14 为周五，最近周一为 2026-08-17，早于 9 月 1 号。
		if out["next"] != "2026-08-17T00:00:00Z" {
			return fmt.Errorf("next=%v", out["next"])
		}
		return nil
	})

	check("每日(析取全覆盖)", func() error {
		resp, body, err := do(http.MethodPost, "/api/cron/next", mkNext("0 0 1-31 * 0-6", "2026-08-14T12:00:00Z"))
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status=%d body=%s", resp.StatusCode, body)
		}
		out, err := parseResp(body)
		if err != nil {
			return err
		}
		if out["next"] != "2026-08-15T00:00:00Z" {
			return fmt.Errorf("next=%v", out["next"])
		}
		return nil
	})

	check("时区偏移保留", func() error {
		resp, body, err := do(http.MethodPost, "/api/cron/next", mkNext("0 0 1 * *", "2026-08-14T12:00:00+08:00"))
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status=%d body=%s", resp.StatusCode, body)
		}
		out, err := parseResp(body)
		if err != nil {
			return err
		}
		if out["next"] != "2026-09-01T00:00:00+08:00" {
			return fmt.Errorf("next=%v", out["next"])
		}
		return nil
	})

	check("不可命中返回错误", func() error {
		resp, body, err := do(http.MethodPost, "/api/cron/next", mkNext("0 0 31 2 *", "2026-08-14T12:00:00Z"))
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusUnprocessableEntity {
			return fmt.Errorf("status=%d body=%s", resp.StatusCode, body)
		}
		out, err := parseResp(body)
		if err != nil {
			return err
		}
		if out["error"] == nil {
			return fmt.Errorf("缺少 error 字段")
		}
		return nil
	})

	check("合法性校验合法", func() error {
		resp, body, err := do(http.MethodPost, "/api/cron/validate", mkExpr("0 0 1 * *"))
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status=%d body=%s", resp.StatusCode, body)
		}
		out, err := parseResp(body)
		if err != nil {
			return err
		}
		if out["valid"] != true {
			return fmt.Errorf("valid=%v", out["valid"])
		}
		return nil
	})

	check("合法性校验非法", func() error {
		resp, body, err := do(http.MethodPost, "/api/cron/validate", mkExpr("60 0 * * *"))
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("status=%d body=%s", resp.StatusCode, body)
		}
		out, err := parseResp(body)
		if err != nil {
			return err
		}
		if out["valid"] != false || out["error"] == nil {
			return fmt.Errorf("valid=%v error=%v", out["valid"], out["error"])
		}
		return nil
	})

	check("字段数错误返回400", func() error {
		resp, body, err := do(http.MethodPost, "/api/cron/parse", mkExpr("0 0 * * * *"))
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusBadRequest {
			return fmt.Errorf("status=%d body=%s", resp.StatusCode, body)
		}
		return nil
	})

	check("越界取值返回400", func() error {
		resp, body, err := do(http.MethodPost, "/api/cron/parse", mkExpr("0 0 0 * *"))
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusBadRequest {
			return fmt.Errorf("status=%d body=%s", resp.StatusCode, body)
		}
		return nil
	})

	check("非法基准时间返回400", func() error {
		resp, body, err := do(http.MethodPost, "/api/cron/next", mkNext("0 0 1 * *", "not-a-time"))
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusBadRequest {
			return fmt.Errorf("status=%d body=%s", resp.StatusCode, body)
		}
		return nil
	})

	check("非法JSON返回400", func() error {
		resp, _, err := do(http.MethodPost, "/api/cron/parse", "{not json")
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusBadRequest {
			return fmt.Errorf("status=%d", resp.StatusCode)
		}
		return nil
	})

	check("多段JSON返回400", func() error {
		resp, _, err := do(http.MethodPost, "/api/cron/parse", mkExpr("0 0 1 * *")+" {}")
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusBadRequest {
			return fmt.Errorf("status=%d", resp.StatusCode)
		}
		return nil
	})

	check("未知字段返回400", func() error {
		resp, _, err := do(http.MethodPost, "/api/cron/parse", `{"foo":"bar"}`)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusBadRequest {
			return fmt.Errorf("status=%d", resp.StatusCode)
		}
		return nil
	})

	fmt.Printf("\n%d passed, %d failed\n", passed, failed)
	if failed > 0 {
		return 1
	}
	return 0
}
