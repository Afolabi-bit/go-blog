package posts

type Handler struct {
	svc *Service
}

func NewHandler(s *Service) *Handler {
	return &Handler{
		svc: s,
	}
}
