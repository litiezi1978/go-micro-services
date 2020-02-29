package frontend

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/harlow/go-micro-services/dialer"
	"github.com/harlow/go-micro-services/registry"
	profilepb "github.com/harlow/go-micro-services/services/profile/proto"
	recommpb "github.com/harlow/go-micro-services/services/recommendation/proto"
	reservepb "github.com/harlow/go-micro-services/services/reservation/proto"
	searchpb "github.com/harlow/go-micro-services/services/search/proto"
	userpb "github.com/harlow/go-micro-services/services/user/proto"
	"github.com/harlow/go-micro-services/tracing"
	"github.com/opentracing/opentracing-go"
	otlog "github.com/opentracing/opentracing-go/log"
)

type Server struct {
	searchClient         searchpb.SearchClient
	profileClient        profilepb.ProfileClient
	recommendationClient recommpb.RecommendationClient
	userClient           userpb.UserClient
	reservationClient    reservepb.ReservationClient
	IpAddr               string
	Port                 int
	Tracer               opentracing.Tracer
	Registry             *registry.Client
}

func (s *Server) Run() error {
	if s.Port == 0 {
		return fmt.Errorf("server port must be set")
	}

	if err := s.initSearchClient("srv-search"); err != nil {
		return err
	}
	if err := s.initProfileClient("srv-profile"); err != nil {
		return err
	}
	if err := s.initRecommendationClient("srv-recommendation"); err != nil {
		return err
	}
	if err := s.initUserClient("srv-user"); err != nil {
		return err
	}
	if err := s.initReservation("srv-reservation"); err != nil {
		return err
	}

	fmt.Printf("frontend before mux\n")
	mux := tracing.NewServeMux(s.Tracer)
	mux.Handle("/", http.FileServer(http.Dir("services/frontend/static")))
	mux.Handle("/hotels", http.HandlerFunc(s.searchHandler))
	mux.Handle("/recommendations", http.HandlerFunc(s.recommendHandler))
	mux.Handle("/user", http.HandlerFunc(s.userHandler))
	mux.Handle("/reservation", http.HandlerFunc(s.reservationHandler))

	fmt.Printf("frontend starts serving\n")
	return http.ListenAndServe(fmt.Sprintf(":%d", s.Port), mux)
}

func (s *Server) initSearchClient(name string) error {
	conn, err := dialer.Dial(
		name,
		dialer.WithTracer(s.Tracer),
		dialer.WithBalancer(s.Registry.Client),
	)
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	s.searchClient = searchpb.NewSearchClient(conn)
	return nil
}

func (s *Server) initProfileClient(name string) error {
	conn, err := dialer.Dial(
		name,
		dialer.WithTracer(s.Tracer),
		dialer.WithBalancer(s.Registry.Client),
	)
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	s.profileClient = profilepb.NewProfileClient(conn)
	return nil
}

func (s *Server) initRecommendationClient(name string) error {
	conn, err := dialer.Dial(
		name,
		dialer.WithTracer(s.Tracer),
		dialer.WithBalancer(s.Registry.Client),
	)
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	s.recommendationClient = recommpb.NewRecommendationClient(conn)
	return nil
}

func (s *Server) initUserClient(name string) error {
	conn, err := dialer.Dial(
		name,
		dialer.WithTracer(s.Tracer),
		dialer.WithBalancer(s.Registry.Client),
	)
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	s.userClient = userpb.NewUserClient(conn)
	return nil
}

func (s *Server) initReservation(name string) error {
	conn, err := dialer.Dial(
		name,
		dialer.WithTracer(s.Tracer),
		dialer.WithBalancer(s.Registry.Client),
	)
	if err != nil {
		return fmt.Errorf("dialer error: %v", err)
	}
	s.reservationClient = reservepb.NewReservationClient(conn)
	return nil
}

func (s *Server) searchHandler(writer http.ResponseWriter, request *http.Request) {
	log.Printf("starts searchHandler\n")
	writer.Header().Set("Access-Control-Allow-Origin", "*")

	span := opentracing.GlobalTracer().StartSpan("/hotels")
	defer span.Finish()
	ctx := opentracing.ContextWithSpan(request.Context(), span)

	//第一步，调用nearby
	inDate, outDate := request.URL.Query().Get("inDate"), request.URL.Query().Get("outDate")
	sLat, sLon := request.URL.Query().Get("lat"), request.URL.Query().Get("lon")
	if inDate == "" || outDate == "" || sLat == "" || sLon == "" {
		http.Error(writer, "Please specify inDate/outDate params", http.StatusBadRequest)
		return
	}
	Lat, _ := strconv.ParseFloat(sLat, 32)
	lat := float32(Lat)
	Lon, _ := strconv.ParseFloat(sLon, 32)
	lon := float32(Lon)

	nearbyReq := searchpb.NearbyRequest{
		Lat:     lat,
		Lon:     lon,
		InDate:  inDate,
		OutDate: outDate,
	}
	span.LogKV("nearByReq", nearbyReq)
	log.Printf("starts searchHandler querying downstream with req=%v\n", nearbyReq)
	searchResp, err := s.searchClient.Nearby(ctx, &nearbyReq)
	if err != nil {
		span.SetTag("error", true)
		span.LogFields(otlog.Error(err))

		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	span.LogKV("nearByResp", searchResp.HotelIds)
	log.Printf("searchHandler received searchResp: %v", searchResp.HotelIds)

	//第二步，调用reserve checkAvailability
	reservReq := reservepb.Request{
		CustomerName: "",
		HotelId:      searchResp.HotelIds,
		InDate:       inDate,
		OutDate:      outDate,
		RoomNumber:   1,
	}
	span.LogKV("ReserveReq", reservReq)
	log.Printf("call reservation client with req=%v\n", reservReq)
	reservationResp, err := s.reservationClient.CheckAvailability(ctx, &reservReq)
	if err != nil {
		span.SetTag("error", true)
		span.LogFields(otlog.Error(err))

		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	span.LogKV("ReservResp", reservationResp.HotelId)
	log.Printf("searchHandler gets reserveResp.HotelId = %s\n", reservationResp.HotelId)

	//第三步，profile
	locale := request.URL.Query().Get("locale")
	if locale == "" {
		locale = "en"
	}
	profileReq := profilepb.Request{
		HotelIds: reservationResp.HotelId,
		Locale:   locale,
	}
	span.LogKV("ProfileReq", profileReq)
	log.Printf("call profile with req=%v\n", profileReq)
	profileResp, err := s.profileClient.GetProfiles(ctx, &profileReq)
	if err != nil {
		span.SetTag("error", true)
		span.LogFields(otlog.Error(err))

		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	span.LogKV("profile response: %v", profileResp.Hotels)
	log.Printf("searchHandler gets profileResp %v\n", profileResp.Hotels)

	json.NewEncoder(writer).Encode(geoJSONResponse(profileResp.Hotels))
}

func (s *Server) recommendHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Access-Control-Allow-Origin", "*")

	span := opentracing.GlobalTracer().StartSpan("/recommendations")
	defer span.Finish()
	ctx := opentracing.ContextWithSpan(request.Context(), span)

	sLat, sLon := request.URL.Query().Get("lat"), request.URL.Query().Get("lon")
	if sLat == "" || sLon == "" {
		span.SetTag("error", true)
		span.LogKV("sLat", "nil", "sLon", "nil")

		http.Error(writer, "Please specify location params", http.StatusBadRequest)
		return
	}
	Lat, _ := strconv.ParseFloat(sLat, 64)
	lat := float64(Lat)
	Lon, _ := strconv.ParseFloat(sLon, 64)
	lon := float64(Lon)

	require := request.URL.Query().Get("require")
	if require != "dis" && require != "rate" && require != "price" {
		span.SetTag("error", true)
		span.LogKV("require", require)

		http.Error(writer, "Please specify require params", http.StatusBadRequest)
		return
	}

	// recommend hotels
	recResp, err := s.recommendationClient.GetRecommendations(ctx, &recommpb.Request{
		Require: require,
		Lat:     float64(lat),
		Lon:     float64(lon),
	})
	if err != nil {
		span.SetTag("error", true)
		span.LogFields(otlog.Error(err))

		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	// grab locale from query params or default to en
	locale := request.URL.Query().Get("locale")
	if locale == "" {
		locale = "en"
	}

	// hotel profiles
	profileResp, err := s.profileClient.GetProfiles(ctx, &profilepb.Request{
		HotelIds: recResp.HotelIds,
		Locale:   locale,
	})
	if err != nil {
		span.SetTag("error", true)
		span.LogFields(otlog.Error(err))
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	json_resp := geoJSONResponse(profileResp.Hotels)
	span.LogKV("response result", json_resp)
	json.NewEncoder(writer).Encode(json_resp)
}

func (s *Server) userHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Access-Control-Allow-Origin", "*")

	span := opentracing.GlobalTracer().StartSpan("/users")
	defer span.Finish()
	ctx := opentracing.ContextWithSpan(request.Context(), span)

	username, password := request.URL.Query().Get("username"), request.URL.Query().Get("password")
	if username == "" || password == "" {
		http.Error(writer, "Please specify username and password", http.StatusBadRequest)
		return
	}

	// Check username and password
	recResp, err := s.userClient.CheckUser(ctx, &userpb.Request{
		Username: username,
		Password: password,
	})
	if err != nil {
		span.SetTag("error", true)
		span.LogFields(otlog.Error(err))
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	str := "Login successfully!"
	if recResp.Correct == false {
		str = "Failed. Please check your username and password. "
	}
	span.LogKV("loginResult", str)

	res := map[string]interface{}{
		"message": str,
	}

	json.NewEncoder(writer).Encode(res)
}

func (s *Server) reservationHandler(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Access-Control-Allow-Origin", "*")

	span := opentracing.GlobalTracer().StartSpan("/reservation")
	defer span.Finish()
	ctx := opentracing.ContextWithSpan(request.Context(), span)

	inDate, outDate := request.URL.Query().Get("inDate"), request.URL.Query().Get("outDate")
	if inDate == "" || outDate == "" {
		http.Error(writer, "Please specify inDate/outDate params", http.StatusBadRequest)
		return
	}

	if !checkDataFormat(inDate) || !checkDataFormat(outDate) {
		http.Error(writer, "Please check inDate/outDate format (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	hotelId := request.URL.Query().Get("hotelId")
	if hotelId == "" {
		http.Error(writer, "Please specify hotelId params", http.StatusBadRequest)
		return
	}

	customerName := request.URL.Query().Get("customerName")
	if customerName == "" {
		http.Error(writer, "Please specify customerName params", http.StatusBadRequest)
		return
	}

	username, password := request.URL.Query().Get("username"), request.URL.Query().Get("password")
	if username == "" || password == "" {
		http.Error(writer, "Please specify username and password", http.StatusBadRequest)
		return
	}

	numberOfRoom := 0
	num := request.URL.Query().Get("number")
	if num != "" {
		numberOfRoom, _ = strconv.Atoi(num)
	}

	// Check username and password
	recResp, err := s.userClient.CheckUser(ctx, &userpb.Request{
		Username: username,
		Password: password,
	})
	if err != nil {
		span.SetTag("error", true)
		span.LogFields(otlog.Error(err))
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	str := "Reserve successfully!"
	if recResp.Correct == false {
		str = "Failed. Please check your username and password. "
	}

	// Make reservation
	resResp, err := s.reservationClient.MakeReservation(ctx, &reservepb.Request{
		CustomerName: customerName,
		HotelId:      []string{hotelId},
		InDate:       inDate,
		OutDate:      outDate,
		RoomNumber:   int32(numberOfRoom),
	})
	if err != nil {
		span.SetTag("error", true)
		span.LogFields(otlog.Error(err))
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(resResp.HotelId) == 0 {
		str = "Failed. Already reserved. "
	}

	res := map[string]interface{}{
		"message": str,
	}

	json.NewEncoder(writer).Encode(res)
}

// return a geoJSON response that allows google map to plot points directly on map
// https://developers.google.com/maps/documentation/javascript/datalayer#sample_geojson
func geoJSONResponse(hs []*profilepb.Hotel) map[string]interface{} {
	fs := []interface{}{}

	for _, h := range hs {
		fs = append(fs, map[string]interface{}{
			"type": "Feature",
			"id":   h.Id,
			"properties": map[string]string{
				"name":         h.Name,
				"phone_number": h.PhoneNumber,
			},
			"geometry": map[string]interface{}{
				"type": "Point",
				"coordinates": []float32{
					h.Address.Lon,
					h.Address.Lat,
				},
			},
		})
	}

	return map[string]interface{}{
		"type":     "FeatureCollection",
		"features": fs,
	}
}

func checkDataFormat(date string) bool {
	if len(date) != 10 {
		return false
	}
	for i := 0; i < 10; i++ {
		if i == 4 || i == 7 {
			if date[i] != '-' {
				return false
			}
		} else {
			if date[i] < '0' || date[i] > '9' {
				return false
			}
		}
	}
	return true
}
